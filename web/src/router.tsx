// Rotas definidas em código (code-based routing do TanStack Router).
// Escolhido em vez do file-based para não depender de um passo extra de
// geração — já temos um (o client da API).
import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  redirect,
} from "@tanstack/react-router";
import { getToken } from "@/lib/api";
import { LoginPage } from "@/pages/LoginPage";
import { FormsListPage } from "@/pages/FormsListPage";
import { FormEditorPage } from "@/pages/FormEditorPage";
import { ResponsesPage } from "@/pages/ResponsesPage";
import { PublicFormPage } from "@/pages/PublicFormPage";

const rootRoute = createRootRoute({ component: () => <Outlet /> });

// Guarda das rotas de admin: sem token, manda para o login.
function requireAuth() {
  if (!getToken()) throw redirect({ to: "/login" });
}

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  beforeLoad: () => {
    throw redirect({ to: getToken() ? "/forms" : "/login" });
  },
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/login",
  component: LoginPage,
});

const formsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/forms",
  beforeLoad: requireAuth,
  component: FormsListPage,
});

const newFormRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/forms/new",
  beforeLoad: requireAuth,
  component: () => <FormEditorPage mode="new" />,
});

const editFormRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/forms/$formId",
  beforeLoad: requireAuth,
  component: () => <FormEditorPage mode="edit" />,
});

const responsesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/forms/$formId/responses",
  beforeLoad: requireAuth,
  component: ResponsesPage,
});

// Rota pública: a URL que o admin compartilha depois de publicar.
const publicFormRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/f/$slug",
  component: PublicFormPage,
});

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  formsRoute,
  newFormRoute,
  editFormRoute,
  responsesRoute,
  publicFormRoute,
]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
