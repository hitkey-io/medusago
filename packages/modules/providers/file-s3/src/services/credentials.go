package services

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type awsCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
}

type credentialsProvider interface {
	retrieve(ctx context.Context) (awsCredentials, error)
}

// staticCredentials are the credentials of the options (access-key).
type staticCredentials awsCredentials

func (s staticCredentials) retrieve(context.Context) (awsCredentials, error) {
	return awsCredentials(s), nil
}

// defaultCredentialsChain mirrors the parts of the SDK default credential
// provider chain used for "s3-iam-role": environment variables, the shared
// credentials file, the ECS container endpoint and the EC2 instance
// metadata service (IMDSv2). Expiring credentials are cached until five
// minutes before they expire.
type defaultCredentialsChain struct {
	mu     sync.Mutex
	cached *awsCredentials
	client *http.Client
}

func newDefaultCredentialsChain() *defaultCredentialsChain {
	return &defaultCredentialsChain{client: &http.Client{Timeout: 5 * time.Second}}
}

func (d *defaultCredentialsChain) retrieve(ctx context.Context) (awsCredentials, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cached != nil && (d.cached.Expiration.IsZero() || time.Until(d.cached.Expiration) > 5*time.Minute) {
		return *d.cached, nil
	}
	sources := []func(context.Context) (awsCredentials, error){d.fromEnv, d.fromSharedFile, d.fromContainer, d.fromInstanceMetadata}
	var errs []error
	for _, src := range sources {
		c, err := src(ctx)
		if err == nil {
			d.cached = &c
			return c, nil
		}
		errs = append(errs, err)
	}
	return awsCredentials{}, fmt.Errorf("Could not load credentials from any providers: %w", errors.Join(errs...))
}

func (d *defaultCredentialsChain) fromEnv(context.Context) (awsCredentials, error) {
	id, secret := os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY")
	if id == "" || secret == "" {
		return awsCredentials{}, errors.New("Unable to find environment variable credentials.")
	}
	c := awsCredentials{AccessKeyID: id, SecretAccessKey: secret, SessionToken: os.Getenv("AWS_SESSION_TOKEN")}
	if exp := os.Getenv("AWS_CREDENTIAL_EXPIRATION"); exp != "" {
		c.Expiration, _ = time.Parse(time.RFC3339, exp)
	}
	return c, nil
}

func (d *defaultCredentialsChain) fromSharedFile(context.Context) (awsCredentials, error) {
	file := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	if file == "" {
		home, _ := os.UserHomeDir()
		file = filepath.Join(home, ".aws", "credentials")
	}
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "default"
	}
	f, err := os.Open(file)
	if err != nil {
		return awsCredentials{}, err
	}
	defer f.Close()
	values := map[string]string{}
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		if section != profile {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			values[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	c := awsCredentials{AccessKeyID: values["aws_access_key_id"], SecretAccessKey: values["aws_secret_access_key"], SessionToken: values["aws_session_token"]}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return awsCredentials{}, fmt.Errorf("Profile %s could not be found or parsed in shared credentials file.", profile)
	}
	return c, nil
}

type metadataCredentials struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Token           string `json:"Token"`
	Expiration      string `json:"Expiration"`
}

func (m metadataCredentials) toCredentials() (awsCredentials, error) {
	if m.AccessKeyID == "" || m.SecretAccessKey == "" {
		return awsCredentials{}, errors.New("invalid credentials response")
	}
	c := awsCredentials{AccessKeyID: m.AccessKeyID, SecretAccessKey: m.SecretAccessKey, SessionToken: m.Token}
	c.Expiration, _ = time.Parse(time.RFC3339, m.Expiration)
	return c, nil
}

func (d *defaultCredentialsChain) getJSON(ctx context.Context, url string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", url, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (d *defaultCredentialsChain) fromContainer(ctx context.Context) (awsCredentials, error) {
	url := ""
	if rel := os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI"); rel != "" {
		url = "http://169.254.170.2" + rel
	} else if full := os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI"); full != "" {
		url = full
	}
	if url == "" {
		return awsCredentials{}, errors.New("no container credentials endpoint")
	}
	headers := map[string]string{}
	if tok := os.Getenv("AWS_CONTAINER_AUTHORIZATION_TOKEN"); tok != "" {
		headers["Authorization"] = tok
	} else if file := os.Getenv("AWS_CONTAINER_AUTHORIZATION_TOKEN_FILE"); file != "" {
		raw, err := os.ReadFile(file)
		if err != nil {
			return awsCredentials{}, err
		}
		headers["Authorization"] = strings.TrimSpace(string(raw))
	}
	var m metadataCredentials
	if err := d.getJSON(ctx, url, headers, &m); err != nil {
		return awsCredentials{}, err
	}
	return m.toCredentials()
}

func (d *defaultCredentialsChain) fromInstanceMetadata(ctx context.Context) (awsCredentials, error) {
	if strings.EqualFold(os.Getenv("AWS_EC2_METADATA_DISABLED"), "true") {
		return awsCredentials{}, errors.New("EC2 instance metadata is disabled")
	}
	base := os.Getenv("AWS_EC2_METADATA_SERVICE_ENDPOINT")
	if base == "" {
		base = "http://169.254.169.254"
	}
	base = strings.TrimSuffix(base, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, base+"/latest/api/token", nil)
	if err != nil {
		return awsCredentials{}, err
	}
	req.Header.Set("x-aws-ec2-metadata-token-ttl-seconds", "21600")
	res, err := d.client.Do(req)
	if err != nil {
		return awsCredentials{}, err
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return awsCredentials{}, fmt.Errorf("instance metadata token: %s", res.Status)
	}
	headers := map[string]string{"x-aws-ec2-metadata-token": string(raw)}
	req, _ = http.NewRequestWithContext(ctx, http.MethodGet, base+"/latest/meta-data/iam/security-credentials/", nil)
	req.Header.Set("x-aws-ec2-metadata-token", string(raw))
	res, err = d.client.Do(req)
	if err != nil {
		return awsCredentials{}, err
	}
	role, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return awsCredentials{}, fmt.Errorf("instance metadata role: %s", res.Status)
	}
	name := strings.TrimSpace(strings.SplitN(string(role), "\n", 2)[0])
	var m metadataCredentials
	if err := d.getJSON(ctx, base+"/latest/meta-data/iam/security-credentials/"+name, headers, &m); err != nil {
		return awsCredentials{}, err
	}
	return m.toCredentials()
}
