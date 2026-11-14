// MedusaGo has no AI assistant: stand-in for @kapaai/react-sdk.
export const useChat = () => ({
  isGeneratingAnswer: false,
  isPreparingAnswer: false,
  submitQuery: (_query?: string) => {},
})
