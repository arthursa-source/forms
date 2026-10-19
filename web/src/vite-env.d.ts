/// <reference types="vite/client" />

interface ImportMetaEnv {
  // Base da API. Em dev fica vazio e o proxy do Vite cuida de /api.
  readonly VITE_API_URL?: string;
}
