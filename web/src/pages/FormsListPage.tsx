import { Link } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, errorMessage } from "@/lib/api";
import { AppShell } from "@/components/AppShell";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

export function FormsListPage() {
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ["forms"],
    queryFn: async () => {
      const { data, error } = await api.GET("/forms");
      if (error) throw error;
      return data;
    },
  });

  // publish e unpublish compartilham a invalidação da lista. Os dois
  // caminhos são escritos literalmente para o openapi-fetch conferir os
  // tipos de cada um.
  const toggle = useMutation({
    mutationFn: async ({ id, action }: { id: number; action: "publish" | "unpublish" }) => {
      const params = { params: { path: { formId: id } } };
      const { data, error } =
        action === "publish"
          ? await api.POST("/forms/{formId}/publish", params)
          : await api.POST("/forms/{formId}/unpublish", params);
      if (error) throw error;
      return data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["forms"] }),
  });

  const remove = useMutation({
    mutationFn: async (id: number) => {
      const { error } = await api.DELETE("/forms/{formId}", { params: { path: { formId: id } } });
      if (error) throw error;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["forms"] }),
  });

  return (
    <AppShell>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Meus formulários</h1>
          <p className="text-sm text-muted-foreground">Crie, edite e publique formulários.</p>
        </div>
        <Button asChild>
          <Link to="/forms/new">Novo formulário</Link>
        </Button>
      </div>

      {isLoading && <p className="text-sm text-muted-foreground">Carregando...</p>}
      {error && <p className="text-sm text-destructive">{errorMessage(error)}</p>}

      {data?.forms.length === 0 && (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            Você ainda não tem formulários. Crie o primeiro.
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        {data?.forms.map((form) => (
          <Card key={form.id}>
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    {form.title}
                    {form.status === "published" ? (
                      <Badge variant="success">publicado</Badge>
                    ) : (
                      <Badge variant="secondary">draft</Badge>
                    )}
                  </CardTitle>
                  {form.description && <CardDescription>{form.description}</CardDescription>}
                  {form.publicPath && (
                    <a
                      href={form.publicPath}
                      target="_blank"
                      rel="noreferrer"
                      className="mt-1 inline-block text-xs text-primary underline"
                    >
                      {window.location.origin}
                      {form.publicPath}
                    </a>
                  )}
                </div>
              </div>
            </CardHeader>
            <CardContent className="flex flex-wrap gap-2 pt-0">
              <Button asChild variant="outline" size="sm">
                <Link to="/forms/$formId" params={{ formId: String(form.id) }}>
                  Editar
                </Link>
              </Button>
              <Button asChild variant="outline" size="sm">
                <Link to="/forms/$formId/responses" params={{ formId: String(form.id) }}>
                  Respostas
                </Link>
              </Button>
              {form.status === "draft" ? (
                <Button
                  size="sm"
                  disabled={toggle.isPending}
                  onClick={() => toggle.mutate({ id: form.id, action: "publish" })}
                >
                  Publicar
                </Button>
              ) : (
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={toggle.isPending}
                  onClick={() => toggle.mutate({ id: form.id, action: "unpublish" })}
                >
                  Despublicar
                </Button>
              )}
              <Button
                size="sm"
                variant="ghost"
                className="text-destructive"
                onClick={() => {
                  if (confirm(`Remover "${form.title}" e todas as respostas?`)) remove.mutate(form.id);
                }}
              >
                Remover
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>

      {toggle.isError && <p className="mt-4 text-sm text-destructive">{errorMessage(toggle.error)}</p>}
    </AppShell>
  );
}
