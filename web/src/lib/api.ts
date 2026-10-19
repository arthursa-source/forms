// Client HTTP da aplicação.
//
// Nada aqui é escrito à mão em termos de tipos: `paths` vem de
// src/api/schema.d.ts, que é GERADO da spec OpenAPI por
// `npm run generate:client` (openapi-typescript). O openapi-fetch usa
// esses tipos para validar caminho, body e resposta em tempo de compilação.
import createClient from "openapi-fetch";
import type { paths } from "@/api/schema";

// Em dev o Vite faz proxy de /api para http://localhost:8080 (ver
// vite.config.ts), então o frontend nunca fala com outra origem.
const baseUrl = import.meta.env.VITE_API_URL ?? "/api";

const TOKEN_KEY = "forms.token";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY);
}

export const api = createClient<paths>({ baseUrl });

// Middleware que anexa o JWT em toda requisição autenticada.
api.use({
  onRequest({ request }) {
    const token = getToken();
    if (token) request.headers.set("Authorization", `Bearer ${token}`);
    return request;
  },
});

// A API devolve erros no formato RFC 7807 (title/detail/errors). Esta
// função transforma isso numa mensagem legível para a tela.
export type ApiErrorShape = {
  detail?: string;
  title?: string;
  errors?: { message?: string; location?: string }[];
};

export function errorMessage(error: unknown, fallback = "Algo deu errado"): string {
  const e = error as ApiErrorShape | undefined;
  if (!e) return fallback;
  const details = e.errors?.map((d) => d.message).filter(Boolean) ?? [];
  if (details.length > 0) return details.join("; ");
  return e.detail ?? e.title ?? fallback;
}

// Erros de validação por campo, para marcar os inputs da página pública.
// O backend manda location = "body.answers.<fieldId>".
export function fieldErrors(error: unknown): Record<string, string> {
  const e = error as ApiErrorShape | undefined;
  const out: Record<string, string> = {};
  for (const d of e?.errors ?? []) {
    const id = d.location?.split(".").pop();
    if (id && d.message) out[id] = d.message;
  }
  return out;
}
