import { CardProps } from "../../components/Card"

type UseSuggestionsProps = {
  keywords: string | string[]
}

// The upstream hook suggests Medusa Cloud pages for some search keywords;
// MedusaGo has no equivalent.
export const useMedusaSuggestions = (_props: UseSuggestionsProps): CardProps | null => null
