--
-- PostgreSQL database dump
--


-- Dumped from database version 15.19
-- Dumped by pg_dump version 15.19

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: account_holder; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: api_key; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.api_key (id, token, salt, redacted, title, type, last_used_at, created_by, created_at, revoked_by, revoked_at, updated_at, deleted_at) VALUES ('apk_01M70HF0X4YJ8S6N7T64D5EKQC', 'pk_49f4488359b159f52ae980744733e3f38b70ed5dd26eb8081d7e525c2be83be2', '', 'pk_49f***be2', 'Default Publishable API Key', 'publishable', NULL, '', '2026-11-09 16:15:06.148+00', NULL, NULL, '2026-11-09 16:15:06.148+00', NULL);


--
-- Data for Name: promotion_campaign; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: promotion; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: promotion_application_method; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: promotion_rule; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: application_method_buy_rules; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: application_method_target_rules; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: auth_identity; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: auth_mfa_factor; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: auth_mfa_recovery_code; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: provider_identity; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: auth_password_reset_token; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: auth_verification; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: payment_collection; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: payment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: capture; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_address; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_line_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_line_item_adjustment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_line_item_tax_line; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_payment_collection; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_promotion; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_shipping_method; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_shipping_method_adjustment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: cart_shipping_method_tax_line; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: credit_line; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: currency; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('usd', '$', '$', 2, 0, '{"value": "0", "precision": 20}', 'US Dollar', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('cad', 'CA$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Canadian Dollar', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('eur', '€', '€', 2, 0, '{"value": "0", "precision": 20}', 'Euro', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('aed', 'AED', 'د.إ.‏', 2, 0, '{"value": "0", "precision": 20}', 'United Arab Emirates Dirham', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('afn', 'Af', '؋', 0, 0, '{"value": "0", "precision": 20}', 'Afghan Afghani', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('all', 'ALL', 'Lek', 0, 0, '{"value": "0", "precision": 20}', 'Albanian Lek', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('amd', 'AMD', 'դր.', 0, 0, '{"value": "0", "precision": 20}', 'Armenian Dram', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('aoa', 'AOA', 'Kz', 2, 0, '{"value": "0", "precision": 20}', 'Angolan Kwanza', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ars', 'AR$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Argentine Peso', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('aud', 'AU$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Australian Dollar', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('awg', 'Afl.', 'Afl.', 2, 0, '{"value": "0", "precision": 20}', 'Aruban Florin', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('azn', 'man.', 'ман.', 2, 0, '{"value": "0", "precision": 20}', 'Azerbaijani Manat', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bam', 'KM', 'KM', 2, 0, '{"value": "0", "precision": 20}', 'Bosnia-Herzegovina Convertible Mark', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bdt', 'Tk', '৳', 2, 0, '{"value": "0", "precision": 20}', 'Bangladeshi Taka', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bgn', 'BGN', 'лв.', 2, 0, '{"value": "0", "precision": 20}', 'Bulgarian Lev', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bhd', 'BD', 'د.ب.‏', 3, 0, '{"value": "0", "precision": 20}', 'Bahraini Dinar', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bif', 'FBu', 'FBu', 0, 0, '{"value": "0", "precision": 20}', 'Burundian Franc', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bnd', 'BN$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Brunei Dollar', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bob', 'Bs', 'Bs', 2, 0, '{"value": "0", "precision": 20}', 'Bolivian Boliviano', '2026-11-09 15:25:03.921+00', '2026-11-09 15:25:03.921+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('brl', 'R$', 'R$', 2, 0, '{"value": "0", "precision": 20}', 'Brazilian Real', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bwp', 'BWP', 'P', 2, 0, '{"value": "0", "precision": 20}', 'Botswanan Pula', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('byn', 'Br', 'руб.', 2, 0, '{"value": "0", "precision": 20}', 'Belarusian Ruble', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('bzd', 'BZ$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Belize Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('cdf', 'CDF', 'FrCD', 2, 0, '{"value": "0", "precision": 20}', 'Congolese Franc', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('chf', 'CHF', 'CHF', 2, 0.05, '{"value": "0.05", "precision": 20}', 'Swiss Franc', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('clp', 'CL$', '$', 0, 0, '{"value": "0", "precision": 20}', 'Chilean Peso', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('cny', 'CN¥', 'CN¥', 2, 0, '{"value": "0", "precision": 20}', 'Chinese Yuan', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('cop', 'CO$', '$', 0, 0, '{"value": "0", "precision": 20}', 'Colombian Peso', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('crc', '₡', '₡', 0, 0, '{"value": "0", "precision": 20}', 'Costa Rican Colón', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('cve', 'CV$', 'CV$', 2, 0, '{"value": "0", "precision": 20}', 'Cape Verdean Escudo', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('czk', 'Kč', 'Kč', 2, 0, '{"value": "0", "precision": 20}', 'Czech Republic Koruna', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('djf', 'Fdj', 'Fdj', 0, 0, '{"value": "0", "precision": 20}', 'Djiboutian Franc', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('dkk', 'Dkr', 'kr', 2, 0, '{"value": "0", "precision": 20}', 'Danish Krone', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('dop', 'RD$', 'RD$', 2, 0, '{"value": "0", "precision": 20}', 'Dominican Peso', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('dzd', 'DA', 'د.ج.‏', 2, 0, '{"value": "0", "precision": 20}', 'Algerian Dinar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('eek', 'Ekr', 'kr', 2, 0, '{"value": "0", "precision": 20}', 'Estonian Kroon', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('egp', 'EGP', 'ج.م.‏', 2, 0, '{"value": "0", "precision": 20}', 'Egyptian Pound', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ern', 'Nfk', 'Nfk', 2, 0, '{"value": "0", "precision": 20}', 'Eritrean Nakfa', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('etb', 'Br', 'Br', 2, 0, '{"value": "0", "precision": 20}', 'Ethiopian Birr', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('gbp', '£', '£', 2, 0, '{"value": "0", "precision": 20}', 'British Pound Sterling', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('gel', 'GEL', 'GEL', 2, 0, '{"value": "0", "precision": 20}', 'Georgian Lari', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ghs', 'GH₵', 'GH₵', 2, 0, '{"value": "0", "precision": 20}', 'Ghanaian Cedi', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('gmd', 'D', 'D', 2, 0, '{"value": "0", "precision": 20}', 'Gambian Dalasi', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('gnf', 'FG', 'FG', 0, 0, '{"value": "0", "precision": 20}', 'Guinean Franc', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('gtq', 'GTQ', 'Q', 2, 0, '{"value": "0", "precision": 20}', 'Guatemalan Quetzal', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('hkd', 'HK$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Hong Kong Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('hnl', 'HNL', 'L', 2, 0, '{"value": "0", "precision": 20}', 'Honduran Lempira', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('hrk', 'kn', 'kn', 2, 0, '{"value": "0", "precision": 20}', 'Croatian Kuna', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('htg', 'HTG', 'G', 2, 0, '{"value": "0", "precision": 20}', 'Haitian Gourde', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('huf', 'Ft', 'Ft', 0, 0, '{"value": "0", "precision": 20}', 'Hungarian Forint', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('idr', 'Rp', 'Rp', 0, 0, '{"value": "0", "precision": 20}', 'Indonesian Rupiah', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ils', '₪', '₪', 2, 0, '{"value": "0", "precision": 20}', 'Israeli New Sheqel', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('inr', 'Rs', '₹', 2, 0, '{"value": "0", "precision": 20}', 'Indian Rupee', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('iqd', 'IQD', 'د.ع.‏', 0, 0, '{"value": "0", "precision": 20}', 'Iraqi Dinar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('irr', 'IRR', '﷼', 0, 0, '{"value": "0", "precision": 20}', 'Iranian Rial', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('irt', 'IRT', 'تومان', 0, 0, '{"value": "0", "precision": 20}', 'Iranian Toman', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('isk', 'Ikr', 'kr', 0, 0, '{"value": "0", "precision": 20}', 'Icelandic Króna', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('jmd', 'J$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Jamaican Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('jod', 'JD', 'د.أ.‏', 3, 0, '{"value": "0", "precision": 20}', 'Jordanian Dinar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('jpy', '¥', '￥', 0, 0, '{"value": "0", "precision": 20}', 'Japanese Yen', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('kes', 'Ksh', 'Ksh', 2, 0, '{"value": "0", "precision": 20}', 'Kenyan Shilling', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('khr', 'KHR', '៛', 2, 0, '{"value": "0", "precision": 20}', 'Cambodian Riel', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('kmf', 'CF', 'FC', 0, 0, '{"value": "0", "precision": 20}', 'Comorian Franc', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('krw', '₩', '₩', 0, 0, '{"value": "0", "precision": 20}', 'South Korean Won', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('kwd', 'KD', 'د.ك.‏', 3, 0, '{"value": "0", "precision": 20}', 'Kuwaiti Dinar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('kzt', 'KZT', 'тңг.', 2, 0, '{"value": "0", "precision": 20}', 'Kazakhstani Tenge', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('lbp', 'LB£', 'ل.ل.‏', 0, 0, '{"value": "0", "precision": 20}', 'Lebanese Pound', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('lkr', 'SLRs', 'SL Re', 2, 0, '{"value": "0", "precision": 20}', 'Sri Lankan Rupee', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ltl', 'Lt', 'Lt', 2, 0, '{"value": "0", "precision": 20}', 'Lithuanian Litas', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('lvl', 'Ls', 'Ls', 2, 0, '{"value": "0", "precision": 20}', 'Latvian Lats', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('lyd', 'LD', 'د.ل.‏', 3, 0, '{"value": "0", "precision": 20}', 'Libyan Dinar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mad', 'MAD', 'د.م.‏', 2, 0, '{"value": "0", "precision": 20}', 'Moroccan Dirham', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mdl', 'MDL', 'MDL', 2, 0, '{"value": "0", "precision": 20}', 'Moldovan Leu', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mga', 'MGA', 'MGA', 0, 0, '{"value": "0", "precision": 20}', 'Malagasy Ariary', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mkd', 'MKD', 'MKD', 2, 0, '{"value": "0", "precision": 20}', 'Macedonian Denar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mmk', 'MMK', 'K', 0, 0, '{"value": "0", "precision": 20}', 'Myanma Kyat', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mnt', 'MNT', '₮', 0, 0, '{"value": "0", "precision": 20}', 'Mongolian Tugrig', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mop', 'MOP$', 'MOP$', 2, 0, '{"value": "0", "precision": 20}', 'Macanese Pataca', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mur', 'MURs', 'MURs', 0, 0, '{"value": "0", "precision": 20}', 'Mauritian Rupee', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mwk', 'K', 'K', 2, 0, '{"value": "0", "precision": 20}', 'Malawian Kwacha', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mxn', 'MX$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Mexican Peso', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('myr', 'RM', 'RM', 2, 0, '{"value": "0", "precision": 20}', 'Malaysian Ringgit', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('mzn', 'MTn', 'MTn', 2, 0, '{"value": "0", "precision": 20}', 'Mozambican Metical', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('nad', 'N$', 'N$', 2, 0, '{"value": "0", "precision": 20}', 'Namibian Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ngn', '₦', '₦', 2, 0, '{"value": "0", "precision": 20}', 'Nigerian Naira', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('nio', 'C$', 'C$', 2, 0, '{"value": "0", "precision": 20}', 'Nicaraguan Córdoba', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('nok', 'Nkr', 'kr', 2, 0, '{"value": "0", "precision": 20}', 'Norwegian Krone', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('npr', 'NPRs', 'नेरू', 2, 0, '{"value": "0", "precision": 20}', 'Nepalese Rupee', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('nzd', 'NZ$', '$', 2, 0, '{"value": "0", "precision": 20}', 'New Zealand Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('omr', 'OMR', 'ر.ع.‏', 3, 0, '{"value": "0", "precision": 20}', 'Omani Rial', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('pab', 'B/.', 'B/.', 2, 0, '{"value": "0", "precision": 20}', 'Panamanian Balboa', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('pen', 'S/.', 'S/.', 2, 0, '{"value": "0", "precision": 20}', 'Peruvian Nuevo Sol', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('php', '₱', '₱', 2, 0, '{"value": "0", "precision": 20}', 'Philippine Peso', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('pkr', 'PKRs', '₨', 0, 0, '{"value": "0", "precision": 20}', 'Pakistani Rupee', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('pln', 'zł', 'zł', 2, 0, '{"value": "0", "precision": 20}', 'Polish Zloty', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('pyg', '₲', '₲', 0, 0, '{"value": "0", "precision": 20}', 'Paraguayan Guarani', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('qar', 'QR', 'ر.ق.‏', 2, 0, '{"value": "0", "precision": 20}', 'Qatari Rial', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ron', 'RON', 'RON', 2, 0, '{"value": "0", "precision": 20}', 'Romanian Leu', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('rsd', 'din.', 'дин.', 0, 0, '{"value": "0", "precision": 20}', 'Serbian Dinar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('rub', 'RUB', '₽.', 2, 0, '{"value": "0", "precision": 20}', 'Russian Ruble', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('rwf', 'RWF', 'FR', 0, 0, '{"value": "0", "precision": 20}', 'Rwandan Franc', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('sar', 'SR', 'ر.س.‏', 2, 0, '{"value": "0", "precision": 20}', 'Saudi Riyal', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('sdg', 'SDG', 'SDG', 2, 0, '{"value": "0", "precision": 20}', 'Sudanese Pound', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('sek', 'Skr', 'kr', 2, 0, '{"value": "0", "precision": 20}', 'Swedish Krona', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('sgd', 'S$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Singapore Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('sos', 'Ssh', 'Ssh', 0, 0, '{"value": "0", "precision": 20}', 'Somali Shilling', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('syp', 'SY£', 'ل.س.‏', 0, 0, '{"value": "0", "precision": 20}', 'Syrian Pound', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('thb', '฿', '฿', 2, 0, '{"value": "0", "precision": 20}', 'Thai Baht', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('tnd', 'DT', 'د.ت.‏', 3, 0, '{"value": "0", "precision": 20}', 'Tunisian Dinar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('top', 'T$', 'T$', 2, 0, '{"value": "0", "precision": 20}', 'Tongan Paʻanga', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('tjs', 'TJS', 'с.', 2, 0, '{"value": "0", "precision": 20}', 'Tajikistani Somoni', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('try', '₺', '₺', 2, 0, '{"value": "0", "precision": 20}', 'Turkish Lira', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ttd', 'TT$', '$', 2, 0, '{"value": "0", "precision": 20}', 'Trinidad and Tobago Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('twd', 'NT$', 'NT$', 2, 0, '{"value": "0", "precision": 20}', 'New Taiwan Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('tzs', 'TSh', 'TSh', 0, 0, '{"value": "0", "precision": 20}', 'Tanzanian Shilling', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('uah', '₴', '₴', 2, 0, '{"value": "0", "precision": 20}', 'Ukrainian Hryvnia', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('ugx', 'USh', 'USh', 0, 0, '{"value": "0", "precision": 20}', 'Ugandan Shilling', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('uyu', '$U', '$', 2, 0, '{"value": "0", "precision": 20}', 'Uruguayan Peso', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('uzs', 'UZS', 'UZS', 0, 0, '{"value": "0", "precision": 20}', 'Uzbekistan Som', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('vef', 'Bs.F.', 'Bs.F.', 2, 0, '{"value": "0", "precision": 20}', 'Venezuelan Bolívar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('vnd', '₫', '₫', 0, 0, '{"value": "0", "precision": 20}', 'Vietnamese Dong', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('xaf', 'FCFA', 'FCFA', 0, 0, '{"value": "0", "precision": 20}', 'CFA Franc BEAC', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('xof', 'CFA', 'CFA', 0, 0, '{"value": "0", "precision": 20}', 'CFA Franc BCEAO', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('xpf', '₣', '₣', 0, 0, '{"value": "0", "precision": 20}', 'CFP Franc', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('yer', 'YR', 'ر.ي.‏', 0, 0, '{"value": "0", "precision": 20}', 'Yemeni Rial', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('zar', 'R', 'R', 2, 0, '{"value": "0", "precision": 20}', 'South African Rand', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('zmk', 'ZK', 'ZK', 0, 0, '{"value": "0", "precision": 20}', 'Zambian Kwacha', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);
INSERT INTO public.currency (code, symbol, symbol_native, decimal_digits, rounding, raw_rounding, name, created_at, updated_at, deleted_at) VALUES ('zwl', 'ZWL$', 'ZWL$', 0, 0, '{"value": "0", "precision": 20}', 'Zimbabwean Dollar', '2026-11-09 15:25:03.922+00', '2026-11-09 15:25:03.922+00', NULL);


--
-- Data for Name: customer; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: customer_account_holder; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: customer_address; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: customer_group; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: customer_group_customer; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: fulfillment_address; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: fulfillment_provider; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.fulfillment_provider (id, is_enabled, created_at, updated_at, deleted_at) VALUES ('manual_manual', true, '2026-11-09 15:25:03.919+00', '2026-11-09 15:25:03.919+00', NULL);


--
-- Data for Name: fulfillment_set; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: service_zone; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: shipping_option_type; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: shipping_profile; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.shipping_profile (id, name, type, metadata, created_at, updated_at, deleted_at) VALUES ('sp_01M70EKFVXJQ9SQF567TDFDJPE', 'Default Shipping Profile', 'default', NULL, '2026-11-09 15:25:06.813+00', '2026-11-09 15:25:06.813+00', NULL);


--
-- Data for Name: shipping_option; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: fulfillment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: fulfillment_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: fulfillment_label; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: geo_zone; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: product_collection; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_collection (id, title, handle, metadata, created_at, updated_at, deleted_at, external_id) VALUES ('pcol_01M70HF0Y6SVSVWYHPB9FJWJBT', 'Summer', 'summer', NULL, '2026-11-09 16:15:06.179324+00', '2026-11-09 16:15:06.179324+00', NULL, NULL);


--
-- Data for Name: product_type; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_type (id, value, metadata, created_at, updated_at, deleted_at, external_id) VALUES ('ptyp_01M70HF0Y3Y1MXP1ZG6RKFZF1C', 'Apparel', NULL, '2026-11-09 16:15:06.179+00', '2026-11-09 16:15:06.179+00', NULL, NULL);


--
-- Data for Name: product; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product (id, title, handle, subtitle, description, is_giftcard, status, thumbnail, weight, length, height, width, origin_country, hs_code, mid_code, material, collection_id, type_id, discountable, external_id, created_at, updated_at, deleted_at, metadata) VALUES ('prod_01M70HF0YS05RTVE5NT0E4WA2X', 'Medusa T-Shirt', 't-shirt', NULL, 'A comfortable cotton t-shirt', false, 'published', 'https://example.com/shirt-1.png', 400, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'pcol_01M70HF0Y6SVSVWYHPB9FJWJBT', 'ptyp_01M70HF0Y3Y1MXP1ZG6RKFZF1C', true, NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, '{"featured": true}');
INSERT INTO public.product (id, title, handle, subtitle, description, is_giftcard, status, thumbnail, weight, length, height, width, origin_country, hs_code, mid_code, material, collection_id, type_id, discountable, external_id, created_at, updated_at, deleted_at, metadata) VALUES ('prod_01M70HF0YSWDNW5ZW5A4Y7341E', 'Medusa Sweatpants', 'sweatpants', NULL, NULL, false, 'draft', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, true, NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product (id, title, handle, subtitle, description, is_giftcard, status, thumbnail, weight, length, height, width, origin_country, hs_code, mid_code, material, collection_id, type_id, discountable, external_id, created_at, updated_at, deleted_at, metadata) VALUES ('prod_01M70HF0YSR70JXNJZ9D08NDVF', 'Discontinued Hoodie', 'hoodie', NULL, NULL, false, 'published', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, true, NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.246+00', '2026-11-09 16:15:06.246+00', NULL);


--
-- Data for Name: image; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.image (id, url, metadata, created_at, updated_at, deleted_at, rank, product_id) VALUES ('img_01M70HF0YVN7PM52RVJ45Z21JS', 'https://example.com/shirt-1.png', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, 0, 'prod_01M70HF0YS05RTVE5NT0E4WA2X');
INSERT INTO public.image (id, url, metadata, created_at, updated_at, deleted_at, rank, product_id) VALUES ('img_01M70HF0YV7CS16P4J5PHFRYVJ', 'https://example.com/shirt-2.png', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, 1, 'prod_01M70HF0YS05RTVE5NT0E4WA2X');


--
-- Data for Name: inventory_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: inventory_level; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: invite; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: invite_rbac_role; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: layout_configuration; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: location_fulfillment_provider; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: location_fulfillment_set; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: notification_provider; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.notification_provider (id, handle, name, is_enabled, channels, created_at, updated_at, deleted_at) VALUES ('local', 'local', 'local', true, '{feed}', '2026-11-09 15:25:03.928+00', '2026-11-09 15:25:03.928+00', NULL);


--
-- Data for Name: notification; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_address; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_cart; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_change; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_change_action; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_claim; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_claim_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_claim_item_image; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_credit_line; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_exchange; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_exchange_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_fulfillment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_line_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_line_item_adjustment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_line_item_tax_line; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_payment_collection; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_promotion; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_shipping; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_shipping_method; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_shipping_method_adjustment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_shipping_method_tax_line; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_summary; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: order_transaction; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: payment_provider; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.payment_provider (id, is_enabled, created_at, updated_at, deleted_at) VALUES ('pp_system_default', true, '2026-11-09 15:25:03.93+00', '2026-11-09 15:25:03.93+00', NULL);


--
-- Data for Name: payment_collection_payment_providers; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: payment_session; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: price_list; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: price_set; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: price; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: price_list_rule; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: price_preference; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.price_preference (id, attribute, value, is_tax_inclusive, created_at, updated_at, deleted_at) VALUES ('prpref_01M70HF0WWSMG93JTXBJMDE1KX', 'currency_code', 'eur', false, '2026-11-09 16:15:06.14+00', '2026-11-09 16:15:06.14+00', NULL);


--
-- Data for Name: price_rule; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: product_category; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_category (id, name, description, handle, mpath, is_active, is_internal, rank, parent_category_id, created_at, updated_at, deleted_at, metadata, external_id) VALUES ('pcat_01M70HF0YE0VS03DWXRC61KMCH', 'Clothing', '', 'clothing', 'pcat_01M70HF0YE0VS03DWXRC61KMCH', true, false, 0, NULL, '2026-11-09 16:15:06.19+00', '2026-11-09 16:15:06.19+00', NULL, NULL, NULL);
INSERT INTO public.product_category (id, name, description, handle, mpath, is_active, is_internal, rank, parent_category_id, created_at, updated_at, deleted_at, metadata, external_id) VALUES ('pcat_01M70HF0YKB61701YM9X6K149R', 'Shirts', '', 'shirts', 'pcat_01M70HF0YE0VS03DWXRC61KMCH.pcat_01M70HF0YKB61701YM9X6K149R', true, false, 0, 'pcat_01M70HF0YE0VS03DWXRC61KMCH', '2026-11-09 16:15:06.195+00', '2026-11-09 16:15:06.195+00', NULL, NULL, NULL);


--
-- Data for Name: product_category_product; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_category_product (product_id, product_category_id) VALUES ('prod_01M70HF0YS05RTVE5NT0E4WA2X', 'pcat_01M70HF0YE0VS03DWXRC61KMCH');


--
-- Data for Name: product_option; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_option (id, title, metadata, created_at, updated_at, deleted_at, is_exclusive) VALUES ('opt_01M70HF0YSPN5JFCQT9Y9VPMRF', 'Size', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, true);
INSERT INTO public.product_option (id, title, metadata, created_at, updated_at, deleted_at, is_exclusive) VALUES ('opt_01M70HF0YSR61J6JB3VFXCSGRN', 'Color', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, true);
INSERT INTO public.product_option (id, title, metadata, created_at, updated_at, deleted_at, is_exclusive) VALUES ('opt_01M70HF0YS41E5XPSA315K99A1', 'Size', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, true);


--
-- Data for Name: product_option_value; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_option_value (id, value, option_id, metadata, created_at, updated_at, deleted_at, rank) VALUES ('optval_01M70HF0YWWD467E3NKX3G8B3X', 'S', 'opt_01M70HF0YSPN5JFCQT9Y9VPMRF', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_option_value (id, value, option_id, metadata, created_at, updated_at, deleted_at, rank) VALUES ('optval_01M70HF0YWG979A8Q4RJPKH6R2', 'Black', 'opt_01M70HF0YSR61J6JB3VFXCSGRN', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_option_value (id, value, option_id, metadata, created_at, updated_at, deleted_at, rank) VALUES ('optval_01M70HF0YWCFW5HBDFNCXEZK3J', 'M', 'opt_01M70HF0YSPN5JFCQT9Y9VPMRF', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_option_value (id, value, option_id, metadata, created_at, updated_at, deleted_at, rank) VALUES ('optval_01M70HF0YWTFBDCX94PT967W2F', 'White', 'opt_01M70HF0YSR61J6JB3VFXCSGRN', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_option_value (id, value, option_id, metadata, created_at, updated_at, deleted_at, rank) VALUES ('optval_01M70HF0YWSK8RFM286GQNWTD6', 'L', 'opt_01M70HF0YSPN5JFCQT9Y9VPMRF', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_option_value (id, value, option_id, metadata, created_at, updated_at, deleted_at, rank) VALUES ('optval_01M70HF0YW362MFCN7JTBWT5J1', 'M', 'opt_01M70HF0YS41E5XPSA315K99A1', NULL, '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);


--
-- Data for Name: product_product_option; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_product_option (id, product_id, product_option_id, created_at, updated_at, deleted_at) VALUES ('prodopt_01M70HF0ZB4F211HVX73YJMASF', 'prod_01M70HF0YS05RTVE5NT0E4WA2X', 'opt_01M70HF0YSPN5JFCQT9Y9VPMRF', '2026-11-09 16:15:06.219+00', '2026-11-09 16:15:06.219+00', NULL);
INSERT INTO public.product_product_option (id, product_id, product_option_id, created_at, updated_at, deleted_at) VALUES ('prodopt_01M70HF0ZBX39J4PCCX1E7K3XS', 'prod_01M70HF0YS05RTVE5NT0E4WA2X', 'opt_01M70HF0YSR61J6JB3VFXCSGRN', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);
INSERT INTO public.product_product_option (id, product_id, product_option_id, created_at, updated_at, deleted_at) VALUES ('prodopt_01M70HF0ZB7RY6TZ3E1PVZVKKB', 'prod_01M70HF0YSWDNW5ZW5A4Y7341E', 'opt_01M70HF0YS41E5XPSA315K99A1', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);


--
-- Data for Name: product_product_option_value; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_product_option_value (id, product_product_option_id, product_option_value_id, created_at, updated_at, deleted_at) VALUES ('prodoptval_01M70HF0ZB5R2ZHA6Y5848HX7Y', 'prodopt_01M70HF0ZB4F211HVX73YJMASF', 'optval_01M70HF0YWWD467E3NKX3G8B3X', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);
INSERT INTO public.product_product_option_value (id, product_product_option_id, product_option_value_id, created_at, updated_at, deleted_at) VALUES ('prodoptval_01M70HF0ZBZ36EZ9Y3A68TERWD', 'prodopt_01M70HF0ZB4F211HVX73YJMASF', 'optval_01M70HF0YWCFW5HBDFNCXEZK3J', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);
INSERT INTO public.product_product_option_value (id, product_product_option_id, product_option_value_id, created_at, updated_at, deleted_at) VALUES ('prodoptval_01M70HF0ZB421QGP7DJ2A7XR54', 'prodopt_01M70HF0ZB4F211HVX73YJMASF', 'optval_01M70HF0YWSK8RFM286GQNWTD6', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);
INSERT INTO public.product_product_option_value (id, product_product_option_id, product_option_value_id, created_at, updated_at, deleted_at) VALUES ('prodoptval_01M70HF0ZB4QH59F612C88JHGJ', 'prodopt_01M70HF0ZBX39J4PCCX1E7K3XS', 'optval_01M70HF0YWG979A8Q4RJPKH6R2', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);
INSERT INTO public.product_product_option_value (id, product_product_option_id, product_option_value_id, created_at, updated_at, deleted_at) VALUES ('prodoptval_01M70HF0ZB55VEP073M8DJFPBG', 'prodopt_01M70HF0ZBX39J4PCCX1E7K3XS', 'optval_01M70HF0YWTFBDCX94PT967W2F', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);
INSERT INTO public.product_product_option_value (id, product_product_option_id, product_option_value_id, created_at, updated_at, deleted_at) VALUES ('prodoptval_01M70HF0ZB73VZZ46YK0793K9W', 'prodopt_01M70HF0ZB7RY6TZ3E1PVZVKKB', 'optval_01M70HF0YW362MFCN7JTBWT5J1', '2026-11-09 16:15:06.22+00', '2026-11-09 16:15:06.22+00', NULL);


--
-- Data for Name: product_sales_channel; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: product_shipping_profile; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: product_tag; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_tag (id, value, metadata, created_at, updated_at, deleted_at, external_id) VALUES ('ptag_01M70HF0YA7BFB05Y7HZ996Q6B', 'cotton', NULL, '2026-11-09 16:15:06.186+00', '2026-11-09 16:15:06.186+00', NULL, NULL);
INSERT INTO public.product_tag (id, value, metadata, created_at, updated_at, deleted_at, external_id) VALUES ('ptag_01M70HF0YA2B5FFBM99SM1TYQT', 'sale', NULL, '2026-11-09 16:15:06.186+00', '2026-11-09 16:15:06.186+00', NULL, NULL);


--
-- Data for Name: product_tags; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_tags (product_id, product_tag_id) VALUES ('prod_01M70HF0YS05RTVE5NT0E4WA2X', 'ptag_01M70HF0YA7BFB05Y7HZ996Q6B');
INSERT INTO public.product_tags (product_id, product_tag_id) VALUES ('prod_01M70HF0YS05RTVE5NT0E4WA2X', 'ptag_01M70HF0YA2B5FFBM99SM1TYQT');
INSERT INTO public.product_tags (product_id, product_tag_id) VALUES ('prod_01M70HF0YSR70JXNJZ9D08NDVF', 'ptag_01M70HF0YA2B5FFBM99SM1TYQT');


--
-- Data for Name: product_variant; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_variant (id, title, sku, barcode, ean, upc, allow_backorder, manage_inventory, hs_code, origin_country, mid_code, material, weight, length, height, width, metadata, variant_rank, product_id, created_at, updated_at, deleted_at, thumbnail) VALUES ('variant_01M70HF0YSNJWSR9T8H26QFKZC', 'S / Black', 'SHIRT-S-BLACK', NULL, NULL, NULL, false, true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 0, 'prod_01M70HF0YS05RTVE5NT0E4WA2X', '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_variant (id, title, sku, barcode, ean, upc, allow_backorder, manage_inventory, hs_code, origin_country, mid_code, material, weight, length, height, width, metadata, variant_rank, product_id, created_at, updated_at, deleted_at, thumbnail) VALUES ('variant_01M70HF0YS46PJZHW302RF0TZF', 'M / Black', 'SHIRT-M-BLACK', NULL, NULL, NULL, false, true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 1, 'prod_01M70HF0YS05RTVE5NT0E4WA2X', '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_variant (id, title, sku, barcode, ean, upc, allow_backorder, manage_inventory, hs_code, origin_country, mid_code, material, weight, length, height, width, metadata, variant_rank, product_id, created_at, updated_at, deleted_at, thumbnail) VALUES ('variant_01M70HF0YSQ7GC76EZ11146M9N', 'M', 'PANTS-M', NULL, NULL, NULL, false, true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 0, 'prod_01M70HF0YSWDNW5ZW5A4Y7341E', '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.205+00', NULL, NULL);
INSERT INTO public.product_variant (id, title, sku, barcode, ean, upc, allow_backorder, manage_inventory, hs_code, origin_country, mid_code, material, weight, length, height, width, metadata, variant_rank, product_id, created_at, updated_at, deleted_at, thumbnail) VALUES ('variant_01M70HF0YS0CHKQKM471XQ9WHN', 'L / White', 'SHIRT-L-WHITE', NULL, NULL, NULL, false, true, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 2, 'prod_01M70HF0YS05RTVE5NT0E4WA2X', '2026-11-09 16:15:06.205+00', '2026-11-09 16:15:06.254+00', '2026-11-09 16:15:06.254+00', NULL);


--
-- Data for Name: product_variant_inventory_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: product_variant_option; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.product_variant_option (variant_id, option_value_id) VALUES ('variant_01M70HF0YSNJWSR9T8H26QFKZC', 'optval_01M70HF0YWWD467E3NKX3G8B3X');
INSERT INTO public.product_variant_option (variant_id, option_value_id) VALUES ('variant_01M70HF0YSNJWSR9T8H26QFKZC', 'optval_01M70HF0YWG979A8Q4RJPKH6R2');
INSERT INTO public.product_variant_option (variant_id, option_value_id) VALUES ('variant_01M70HF0YSQ7GC76EZ11146M9N', 'optval_01M70HF0YW362MFCN7JTBWT5J1');
INSERT INTO public.product_variant_option (variant_id, option_value_id) VALUES ('variant_01M70HF0YS46PJZHW302RF0TZF', 'optval_01M70HF0YWCFW5HBDFNCXEZK3J');
INSERT INTO public.product_variant_option (variant_id, option_value_id) VALUES ('variant_01M70HF0YS46PJZHW302RF0TZF', 'optval_01M70HF0YWG979A8Q4RJPKH6R2');
INSERT INTO public.product_variant_option (variant_id, option_value_id) VALUES ('variant_01M70HF0YS0CHKQKM471XQ9WHN', 'optval_01M70HF0YWSK8RFM286GQNWTD6');
INSERT INTO public.product_variant_option (variant_id, option_value_id) VALUES ('variant_01M70HF0YS0CHKQKM471XQ9WHN', 'optval_01M70HF0YWTFBDCX94PT967W2F');


--
-- Data for Name: product_variant_price_set; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: product_variant_product_image; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: promotion_campaign_budget; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: promotion_campaign_budget_usage; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: promotion_promotion_rule; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: promotion_rule_value; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: property_label; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: publishable_api_key_sales_channel; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.publishable_api_key_sales_channel (publishable_key_id, sales_channel_id, id, created_at, updated_at, deleted_at) VALUES ('apk_01M70HF0X4YJ8S6N7T64D5EKQC', 'sc_01M70HF0W0C51DQCADBW28N58Y', 'pksc_01M70HF0XA2QRX53S8HNXX70TN', '2026-11-09 16:15:06.151676+00', '2026-11-09 16:15:06.151676+00', NULL);


--
-- Data for Name: refund; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: refund_reason; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.refund_reason (id, label, description, metadata, created_at, updated_at, deleted_at, code) VALUES ('refr_01M70EKAE87Q8X6SZV9XQN0DX0', 'Shipping Issue', 'Refund due to lost, delayed, or misdelivered shipment', NULL, '2026-11-09 15:25:01.146769+00', '2026-11-09 15:25:01.146769+00', NULL, 'shipping_issue');
INSERT INTO public.refund_reason (id, label, description, metadata, created_at, updated_at, deleted_at, code) VALUES ('refr_01M70EKAE8E6PD5FEGE92WQ5DR', 'Customer Care Adjustment', 'Refund given as goodwill or compensation for inconvenience', NULL, '2026-11-09 15:25:01.146769+00', '2026-11-09 15:25:01.146769+00', NULL, 'customer_care_adjustment');
INSERT INTO public.refund_reason (id, label, description, metadata, created_at, updated_at, deleted_at, code) VALUES ('refr_01M70EKAE8QNKJBDB1RPEQVKNB', 'Pricing Error', 'Refund to correct an overcharge, missing discount, or incorrect price', NULL, '2026-11-09 15:25:01.146769+00', '2026-11-09 15:25:01.146769+00', NULL, 'pricing_error');


--
-- Data for Name: region; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.region (id, name, currency_code, metadata, created_at, updated_at, deleted_at, automatic_taxes) VALUES ('reg_01M70HF0XGPQK7APR05NE9JKBJ', 'Europe', 'eur', '{"zone": "eu"}', '2026-11-09 16:15:06.164+00', '2026-11-09 16:15:06.164+00', NULL, true);
INSERT INTO public.region (id, name, currency_code, metadata, created_at, updated_at, deleted_at, automatic_taxes) VALUES ('reg_01M70HF0XG7RHQ3B2CE4G5YG0X', 'North America', 'usd', NULL, '2026-11-09 16:15:06.164+00', '2026-11-09 16:15:06.164+00', NULL, false);
INSERT INTO public.region (id, name, currency_code, metadata, created_at, updated_at, deleted_at, automatic_taxes) VALUES ('reg_01M70HF0XH4AWRH9KK78R7H1HE', 'Archived', 'gbp', NULL, '2026-11-09 16:15:06.164+00', '2026-11-09 16:15:06.173+00', '2026-11-09 16:15:06.172+00', true);


--
-- Data for Name: region_country; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('af', 'afg', '004', 'AFGHANISTAN', 'Afghanistan', NULL, NULL, '2026-11-09 15:25:03.901+00', '2026-11-09 15:25:03.901+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('al', 'alb', '008', 'ALBANIA', 'Albania', NULL, NULL, '2026-11-09 15:25:03.901+00', '2026-11-09 15:25:03.901+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('dz', 'dza', '012', 'ALGERIA', 'Algeria', NULL, NULL, '2026-11-09 15:25:03.901+00', '2026-11-09 15:25:03.901+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('as', 'asm', '016', 'AMERICAN SAMOA', 'American Samoa', NULL, NULL, '2026-11-09 15:25:03.901+00', '2026-11-09 15:25:03.901+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ad', 'and', '020', 'ANDORRA', 'Andorra', NULL, NULL, '2026-11-09 15:25:03.901+00', '2026-11-09 15:25:03.901+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ao', 'ago', '024', 'ANGOLA', 'Angola', NULL, NULL, '2026-11-09 15:25:03.901+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ai', 'aia', '660', 'ANGUILLA', 'Anguilla', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('aq', 'ata', '010', 'ANTARCTICA', 'Antarctica', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ag', 'atg', '028', 'ANTIGUA AND BARBUDA', 'Antigua and Barbuda', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ar', 'arg', '032', 'ARGENTINA', 'Argentina', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('am', 'arm', '051', 'ARMENIA', 'Armenia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('aw', 'abw', '533', 'ARUBA', 'Aruba', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('au', 'aus', '036', 'AUSTRALIA', 'Australia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('at', 'aut', '040', 'AUSTRIA', 'Austria', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('az', 'aze', '031', 'AZERBAIJAN', 'Azerbaijan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bs', 'bhs', '044', 'BAHAMAS', 'Bahamas', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bh', 'bhr', '048', 'BAHRAIN', 'Bahrain', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bd', 'bgd', '050', 'BANGLADESH', 'Bangladesh', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bb', 'brb', '052', 'BARBADOS', 'Barbados', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('by', 'blr', '112', 'BELARUS', 'Belarus', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('be', 'bel', '056', 'BELGIUM', 'Belgium', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bz', 'blz', '084', 'BELIZE', 'Belize', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bj', 'ben', '204', 'BENIN', 'Benin', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bm', 'bmu', '060', 'BERMUDA', 'Bermuda', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bt', 'btn', '064', 'BHUTAN', 'Bhutan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bo', 'bol', '068', 'BOLIVIA', 'Bolivia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bq', 'bes', '535', 'BONAIRE, SINT EUSTATIUS AND SABA', 'Bonaire, Sint Eustatius and Saba', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ba', 'bih', '070', 'BOSNIA AND HERZEGOVINA', 'Bosnia and Herzegovina', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bw', 'bwa', '072', 'BOTSWANA', 'Botswana', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bv', 'bvd', '074', 'BOUVET ISLAND', 'Bouvet Island', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('br', 'bra', '076', 'BRAZIL', 'Brazil', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('io', 'iot', '086', 'BRITISH INDIAN OCEAN TERRITORY', 'British Indian Ocean Territory', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bn', 'brn', '096', 'BRUNEI DARUSSALAM', 'Brunei Darussalam', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bg', 'bgr', '100', 'BULGARIA', 'Bulgaria', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bf', 'bfa', '854', 'BURKINA FASO', 'Burkina Faso', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bi', 'bdi', '108', 'BURUNDI', 'Burundi', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('kh', 'khm', '116', 'CAMBODIA', 'Cambodia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cm', 'cmr', '120', 'CAMEROON', 'Cameroon', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cv', 'cpv', '132', 'CAPE VERDE', 'Cape Verde', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ky', 'cym', '136', 'CAYMAN ISLANDS', 'Cayman Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cf', 'caf', '140', 'CENTRAL AFRICAN REPUBLIC', 'Central African Republic', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('td', 'tcd', '148', 'CHAD', 'Chad', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cl', 'chl', '152', 'CHILE', 'Chile', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cn', 'chn', '156', 'CHINA', 'China', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cx', 'cxr', '162', 'CHRISTMAS ISLAND', 'Christmas Island', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cc', 'cck', '166', 'COCOS (KEELING) ISLANDS', 'Cocos (Keeling) Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('co', 'col', '170', 'COLOMBIA', 'Colombia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('km', 'com', '174', 'COMOROS', 'Comoros', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cg', 'cog', '178', 'CONGO', 'Congo', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cd', 'cod', '180', 'CONGO, THE DEMOCRATIC REPUBLIC OF THE', 'Congo, the Democratic Republic of the', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ck', 'cok', '184', 'COOK ISLANDS', 'Cook Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cr', 'cri', '188', 'COSTA RICA', 'Costa Rica', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ci', 'civ', '384', 'COTE D''IVOIRE', 'Cote D''Ivoire', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('hr', 'hrv', '191', 'CROATIA', 'Croatia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cu', 'cub', '192', 'CUBA', 'Cuba', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cw', 'cuw', '531', 'CURAÇAO', 'Curaçao', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cy', 'cyp', '196', 'CYPRUS', 'Cyprus', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('cz', 'cze', '203', 'CZECH REPUBLIC', 'Czech Republic', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('dk', 'dnk', '208', 'DENMARK', 'Denmark', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('dj', 'dji', '262', 'DJIBOUTI', 'Djibouti', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('dm', 'dma', '212', 'DOMINICA', 'Dominica', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('do', 'dom', '214', 'DOMINICAN REPUBLIC', 'Dominican Republic', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ec', 'ecu', '218', 'ECUADOR', 'Ecuador', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('eg', 'egy', '818', 'EGYPT', 'Egypt', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sv', 'slv', '222', 'EL SALVADOR', 'El Salvador', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gq', 'gnq', '226', 'EQUATORIAL GUINEA', 'Equatorial Guinea', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('er', 'eri', '232', 'ERITREA', 'Eritrea', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ee', 'est', '233', 'ESTONIA', 'Estonia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('et', 'eth', '231', 'ETHIOPIA', 'Ethiopia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('fk', 'flk', '238', 'FALKLAND ISLANDS (MALVINAS)', 'Falkland Islands (Malvinas)', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('fo', 'fro', '234', 'FAROE ISLANDS', 'Faroe Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('fj', 'fji', '242', 'FIJI', 'Fiji', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('fi', 'fin', '246', 'FINLAND', 'Finland', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gf', 'guf', '254', 'FRENCH GUIANA', 'French Guiana', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pf', 'pyf', '258', 'FRENCH POLYNESIA', 'French Polynesia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tf', 'atf', '260', 'FRENCH SOUTHERN TERRITORIES', 'French Southern Territories', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ga', 'gab', '266', 'GABON', 'Gabon', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gm', 'gmb', '270', 'GAMBIA', 'Gambia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ge', 'geo', '268', 'GEORGIA', 'Georgia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gh', 'gha', '288', 'GHANA', 'Ghana', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gi', 'gib', '292', 'GIBRALTAR', 'Gibraltar', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gr', 'grc', '300', 'GREECE', 'Greece', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gl', 'grl', '304', 'GREENLAND', 'Greenland', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gd', 'grd', '308', 'GRENADA', 'Grenada', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gp', 'glp', '312', 'GUADELOUPE', 'Guadeloupe', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gu', 'gum', '316', 'GUAM', 'Guam', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gt', 'gtm', '320', 'GUATEMALA', 'Guatemala', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gg', 'ggy', '831', 'GUERNSEY', 'Guernsey', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gn', 'gin', '324', 'GUINEA', 'Guinea', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gw', 'gnb', '624', 'GUINEA-BISSAU', 'Guinea-Bissau', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gy', 'guy', '328', 'GUYANA', 'Guyana', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ht', 'hti', '332', 'HAITI', 'Haiti', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('hm', 'hmd', '334', 'HEARD ISLAND AND MCDONALD ISLANDS', 'Heard Island And Mcdonald Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('va', 'vat', '336', 'HOLY SEE (VATICAN CITY STATE)', 'Holy See (Vatican City State)', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('hn', 'hnd', '340', 'HONDURAS', 'Honduras', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('hk', 'hkg', '344', 'HONG KONG', 'Hong Kong', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('hu', 'hun', '348', 'HUNGARY', 'Hungary', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('is', 'isl', '352', 'ICELAND', 'Iceland', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('in', 'ind', '356', 'INDIA', 'India', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('id', 'idn', '360', 'INDONESIA', 'Indonesia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ir', 'irn', '364', 'IRAN, ISLAMIC REPUBLIC OF', 'Iran, Islamic Republic of', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('iq', 'irq', '368', 'IRAQ', 'Iraq', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ie', 'irl', '372', 'IRELAND', 'Ireland', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('im', 'imn', '833', 'ISLE OF MAN', 'Isle Of Man', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('il', 'isr', '376', 'ISRAEL', 'Israel', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('jm', 'jam', '388', 'JAMAICA', 'Jamaica', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('jp', 'jpn', '392', 'JAPAN', 'Japan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('je', 'jey', '832', 'JERSEY', 'Jersey', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('jo', 'jor', '400', 'JORDAN', 'Jordan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('kz', 'kaz', '398', 'KAZAKHSTAN', 'Kazakhstan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ke', 'ken', '404', 'KENYA', 'Kenya', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ki', 'kir', '296', 'KIRIBATI', 'Kiribati', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('kp', 'prk', '408', 'KOREA, DEMOCRATIC PEOPLE''S REPUBLIC OF', 'Korea, Democratic People''s Republic of', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('kr', 'kor', '410', 'KOREA, REPUBLIC OF', 'Korea, Republic of', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('xk', 'xkx', '900', 'KOSOVO', 'Kosovo', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('kw', 'kwt', '414', 'KUWAIT', 'Kuwait', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('kg', 'kgz', '417', 'KYRGYZSTAN', 'Kyrgyzstan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('la', 'lao', '418', 'LAO PEOPLE''S DEMOCRATIC REPUBLIC', 'Lao People''s Democratic Republic', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('lv', 'lva', '428', 'LATVIA', 'Latvia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('lb', 'lbn', '422', 'LEBANON', 'Lebanon', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ls', 'lso', '426', 'LESOTHO', 'Lesotho', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('lr', 'lbr', '430', 'LIBERIA', 'Liberia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ly', 'lby', '434', 'LIBYA', 'Libya', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('li', 'lie', '438', 'LIECHTENSTEIN', 'Liechtenstein', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('lt', 'ltu', '440', 'LITHUANIA', 'Lithuania', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('lu', 'lux', '442', 'LUXEMBOURG', 'Luxembourg', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mo', 'mac', '446', 'MACAO', 'Macao', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mg', 'mdg', '450', 'MADAGASCAR', 'Madagascar', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mw', 'mwi', '454', 'MALAWI', 'Malawi', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('my', 'mys', '458', 'MALAYSIA', 'Malaysia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mv', 'mdv', '462', 'MALDIVES', 'Maldives', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ml', 'mli', '466', 'MALI', 'Mali', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mt', 'mlt', '470', 'MALTA', 'Malta', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mh', 'mhl', '584', 'MARSHALL ISLANDS', 'Marshall Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mq', 'mtq', '474', 'MARTINIQUE', 'Martinique', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mr', 'mrt', '478', 'MAURITANIA', 'Mauritania', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mu', 'mus', '480', 'MAURITIUS', 'Mauritius', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('yt', 'myt', '175', 'MAYOTTE', 'Mayotte', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mx', 'mex', '484', 'MEXICO', 'Mexico', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('fm', 'fsm', '583', 'MICRONESIA, FEDERATED STATES OF', 'Micronesia, Federated States of', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('md', 'mda', '498', 'MOLDOVA, REPUBLIC OF', 'Moldova, Republic of', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mc', 'mco', '492', 'MONACO', 'Monaco', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mn', 'mng', '496', 'MONGOLIA', 'Mongolia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('me', 'mne', '499', 'MONTENEGRO', 'Montenegro', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ms', 'msr', '500', 'MONTSERRAT', 'Montserrat', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ma', 'mar', '504', 'MOROCCO', 'Morocco', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mz', 'moz', '508', 'MOZAMBIQUE', 'Mozambique', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mm', 'mmr', '104', 'MYANMAR', 'Myanmar', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('na', 'nam', '516', 'NAMIBIA', 'Namibia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('nr', 'nru', '520', 'NAURU', 'Nauru', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('np', 'npl', '524', 'NEPAL', 'Nepal', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('nl', 'nld', '528', 'NETHERLANDS', 'Netherlands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('nc', 'ncl', '540', 'NEW CALEDONIA', 'New Caledonia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('nz', 'nzl', '554', 'NEW ZEALAND', 'New Zealand', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ni', 'nic', '558', 'NICARAGUA', 'Nicaragua', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ne', 'ner', '562', 'NIGER', 'Niger', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ng', 'nga', '566', 'NIGERIA', 'Nigeria', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('nu', 'niu', '570', 'NIUE', 'Niue', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('nf', 'nfk', '574', 'NORFOLK ISLAND', 'Norfolk Island', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mk', 'mkd', '807', 'NORTH MACEDONIA', 'North Macedonia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mp', 'mnp', '580', 'NORTHERN MARIANA ISLANDS', 'Northern Mariana Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('no', 'nor', '578', 'NORWAY', 'Norway', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('om', 'omn', '512', 'OMAN', 'Oman', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pk', 'pak', '586', 'PAKISTAN', 'Pakistan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pw', 'plw', '585', 'PALAU', 'Palau', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ps', 'pse', '275', 'PALESTINIAN TERRITORY, OCCUPIED', 'Palestinian Territory, Occupied', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pa', 'pan', '591', 'PANAMA', 'Panama', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pg', 'png', '598', 'PAPUA NEW GUINEA', 'Papua New Guinea', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('py', 'pry', '600', 'PARAGUAY', 'Paraguay', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pe', 'per', '604', 'PERU', 'Peru', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ph', 'phl', '608', 'PHILIPPINES', 'Philippines', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pn', 'pcn', '612', 'PITCAIRN', 'Pitcairn', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pl', 'pol', '616', 'POLAND', 'Poland', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pt', 'prt', '620', 'PORTUGAL', 'Portugal', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pr', 'pri', '630', 'PUERTO RICO', 'Puerto Rico', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('qa', 'qat', '634', 'QATAR', 'Qatar', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('re', 'reu', '638', 'REUNION', 'Reunion', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ro', 'rom', '642', 'ROMANIA', 'Romania', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ru', 'rus', '643', 'RUSSIAN FEDERATION', 'Russian Federation', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('rw', 'rwa', '646', 'RWANDA', 'Rwanda', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('bl', 'blm', '652', 'SAINT BARTHÉLEMY', 'Saint Barthélemy', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sh', 'shn', '654', 'SAINT HELENA', 'Saint Helena', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('kn', 'kna', '659', 'SAINT KITTS AND NEVIS', 'Saint Kitts and Nevis', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('lc', 'lca', '662', 'SAINT LUCIA', 'Saint Lucia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('mf', 'maf', '663', 'SAINT MARTIN (FRENCH PART)', 'Saint Martin (French part)', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('pm', 'spm', '666', 'SAINT PIERRE AND MIQUELON', 'Saint Pierre and Miquelon', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('vc', 'vct', '670', 'SAINT VINCENT AND THE GRENADINES', 'Saint Vincent and the Grenadines', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ws', 'wsm', '882', 'SAMOA', 'Samoa', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sm', 'smr', '674', 'SAN MARINO', 'San Marino', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('st', 'stp', '678', 'SAO TOME AND PRINCIPE', 'Sao Tome and Principe', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sa', 'sau', '682', 'SAUDI ARABIA', 'Saudi Arabia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sn', 'sen', '686', 'SENEGAL', 'Senegal', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('rs', 'srb', '688', 'SERBIA', 'Serbia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sc', 'syc', '690', 'SEYCHELLES', 'Seychelles', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sl', 'sle', '694', 'SIERRA LEONE', 'Sierra Leone', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sg', 'sgp', '702', 'SINGAPORE', 'Singapore', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sx', 'sxm', '534', 'SINT MAARTEN', 'Sint Maarten', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sk', 'svk', '703', 'SLOVAKIA', 'Slovakia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('si', 'svn', '705', 'SLOVENIA', 'Slovenia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sb', 'slb', '090', 'SOLOMON ISLANDS', 'Solomon Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('so', 'som', '706', 'SOMALIA', 'Somalia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('za', 'zaf', '710', 'SOUTH AFRICA', 'South Africa', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gs', 'sgs', '239', 'SOUTH GEORGIA AND THE SOUTH SANDWICH ISLANDS', 'South Georgia and the South Sandwich Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ss', 'ssd', '728', 'SOUTH SUDAN', 'South Sudan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('es', 'esp', '724', 'SPAIN', 'Spain', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('lk', 'lka', '144', 'SRI LANKA', 'Sri Lanka', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sd', 'sdn', '729', 'SUDAN', 'Sudan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sr', 'sur', '740', 'SURINAME', 'Suriname', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sj', 'sjm', '744', 'SVALBARD AND JAN MAYEN', 'Svalbard and Jan Mayen', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sz', 'swz', '748', 'SWAZILAND', 'Swaziland', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('se', 'swe', '752', 'SWEDEN', 'Sweden', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ch', 'che', '756', 'SWITZERLAND', 'Switzerland', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('sy', 'syr', '760', 'SYRIAN ARAB REPUBLIC', 'Syrian Arab Republic', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tw', 'twn', '158', 'TAIWAN, PROVINCE OF CHINA', 'Taiwan, Province of China', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tj', 'tjk', '762', 'TAJIKISTAN', 'Tajikistan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tz', 'tza', '834', 'TANZANIA, UNITED REPUBLIC OF', 'Tanzania, United Republic of', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('th', 'tha', '764', 'THAILAND', 'Thailand', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tl', 'tls', '626', 'TIMOR LESTE', 'Timor Leste', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tg', 'tgo', '768', 'TOGO', 'Togo', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tk', 'tkl', '772', 'TOKELAU', 'Tokelau', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('to', 'ton', '776', 'TONGA', 'Tonga', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tt', 'tto', '780', 'TRINIDAD AND TOBAGO', 'Trinidad and Tobago', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tn', 'tun', '788', 'TUNISIA', 'Tunisia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tr', 'tur', '792', 'TURKEY', 'Turkey', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tm', 'tkm', '795', 'TURKMENISTAN', 'Turkmenistan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tc', 'tca', '796', 'TURKS AND CAICOS ISLANDS', 'Turks and Caicos Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('tv', 'tuv', '798', 'TUVALU', 'Tuvalu', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ug', 'uga', '800', 'UGANDA', 'Uganda', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ua', 'ukr', '804', 'UKRAINE', 'Ukraine', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ae', 'are', '784', 'UNITED ARAB EMIRATES', 'United Arab Emirates', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('um', 'umi', '581', 'UNITED STATES MINOR OUTLYING ISLANDS', 'United States Minor Outlying Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('uy', 'ury', '858', 'URUGUAY', 'Uruguay', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('uz', 'uzb', '860', 'UZBEKISTAN', 'Uzbekistan', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('vu', 'vut', '548', 'VANUATU', 'Vanuatu', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ve', 'ven', '862', 'VENEZUELA', 'Venezuela', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('vn', 'vnm', '704', 'VIET NAM', 'Viet Nam', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('vg', 'vgb', '092', 'VIRGIN ISLANDS, BRITISH', 'Virgin Islands, British', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('vi', 'vir', '850', 'VIRGIN ISLANDS, U.S.', 'Virgin Islands, U.S.', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('wf', 'wlf', '876', 'WALLIS AND FUTUNA', 'Wallis and Futuna', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('eh', 'esh', '732', 'WESTERN SAHARA', 'Western Sahara', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ye', 'yem', '887', 'YEMEN', 'Yemen', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('zm', 'zmb', '894', 'ZAMBIA', 'Zambia', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('zw', 'zwe', '716', 'ZIMBABWE', 'Zimbabwe', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ax', 'ala', '248', 'ÅLAND ISLANDS', 'Åland Islands', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 15:25:03.902+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('ca', 'can', '124', 'CANADA', 'Canada', 'reg_01M70HF0XG7RHQ3B2CE4G5YG0X', NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 16:15:06.164+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('fr', 'fra', '250', 'FRANCE', 'France', 'reg_01M70HF0XGPQK7APR05NE9JKBJ', NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 16:15:06.164+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('de', 'deu', '276', 'GERMANY', 'Germany', 'reg_01M70HF0XGPQK7APR05NE9JKBJ', NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 16:15:06.164+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('it', 'ita', '380', 'ITALY', 'Italy', 'reg_01M70HF0XGPQK7APR05NE9JKBJ', NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 16:15:06.164+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('us', 'usa', '840', 'UNITED STATES', 'United States', 'reg_01M70HF0XG7RHQ3B2CE4G5YG0X', NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 16:15:06.164+00', NULL);
INSERT INTO public.region_country (iso_2, iso_3, num_code, name, display_name, region_id, metadata, created_at, updated_at, deleted_at) VALUES ('gb', 'gbr', '826', 'UNITED KINGDOM', 'United Kingdom', NULL, NULL, '2026-11-09 15:25:03.902+00', '2026-11-09 16:15:06.176+00', NULL);


--
-- Data for Name: region_payment_provider; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: reservation_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: return; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: return_fulfillment; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: return_item; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: return_reason; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: sales_channel; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.sales_channel (id, name, description, is_disabled, metadata, created_at, updated_at, deleted_at) VALUES ('sc_01M70HF0W0C51DQCADBW28N58Y', 'Default Sales Channel', 'Created by Medusa', false, NULL, '2026-11-09 16:15:06.112+00', '2026-11-09 16:15:06.112+00', NULL);


--
-- Data for Name: sales_channel_stock_location; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: search_index; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: search_index_version; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: search_index_sync; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: search_postgres_index; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: shipping_option_price_set; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: shipping_option_rule; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: stock_location_address; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: stock_location; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: store; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.store (id, name, default_sales_channel_id, default_region_id, default_location_id, metadata, created_at, updated_at, deleted_at) VALUES ('store_01M70HF0WGCBX0JRWJECFHR6AS', 'Medusa Store', 'sc_01M70HF0W0C51DQCADBW28N58Y', NULL, NULL, NULL, '2026-11-09 16:15:06.124859+00', '2026-11-09 16:15:06.124859+00', NULL);


--
-- Data for Name: store_currency; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.store_currency (id, currency_code, is_default, store_id, created_at, updated_at, deleted_at) VALUES ('stocur_01M70HF0WJ5ZS0TZBVNEST4MGR', 'eur', true, 'store_01M70HF0WGCBX0JRWJECFHR6AS', '2026-11-09 16:15:06.124859+00', '2026-11-09 16:15:06.124859+00', NULL);


--
-- Data for Name: store_locale; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: tax_provider; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.tax_provider (id, is_enabled, created_at, updated_at, deleted_at) VALUES ('tp_system', true, '2026-11-09 15:25:03.931+00', '2026-11-09 15:25:03.931+00', NULL);


--
-- Data for Name: tax_region; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: tax_rate; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: tax_rate_rule; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user_preference; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user_rbac_role; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: view_configuration; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: workflow_execution; Type: TABLE DATA; Schema: public; Owner: -
--



--
--



--
--



--
-- Name: order_change_action_ordering_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.order_change_action_ordering_seq', 1, false);


--
-- Name: order_claim_display_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.order_claim_display_id_seq', 1, false);


--
-- Name: order_display_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.order_display_id_seq', 1, false);


--
-- Name: order_exchange_display_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.order_exchange_display_id_seq', 1, false);


--
-- Name: return_display_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.return_display_id_seq', 1, false);


--
--



--
-- PostgreSQL database dump complete
--


