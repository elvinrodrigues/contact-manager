/// <reference types="vite/client" />

interface ImportMetaEnv {
  /**
   * Public origin of the backend API, e.g. https://contact-manager-api.onrender.com
   * Leave unset in development so requests go to "/api" and Vite's proxy handles them.
   */
  readonly VITE_API_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
