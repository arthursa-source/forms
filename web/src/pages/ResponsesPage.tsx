import { Link, useParams } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { api, errorMessage } from "@/lib/api";
import { AppShell } from "@/components/AppShell";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function ResponsesPage() {
  const { formId } = useParams({ from: "/forms/$formId/responses" });
  const id = Number(formId);

  const form = useQuery({
    queryKey: ["forms", id],
    queryFn: async () => {
      const { data, error } = await api.GET("/forms/{formId}", { params: { path: { formId: id } } });
      if (error) throw error;
      return data;
    },
  });

  const responses = useQuery({
    queryKey: ["forms", id, "responses"],
    queryFn: async () => {
      const { data, error } = await api.GET("/forms/{formId}/responses", {
        params: { path: { formId: id } },
      });
      if (error) throw error;
      return data;
    },
  });

  return (
    <AppShell>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Respostas</h1>
          <p className="text-sm text-muted-foreground">
            {form.data?.title ?? "..."} — {responses.data?.total ?? 0} resposta(s)
          </p>
        </div>
        <Button asChild variant="outline">
          <Link to="/forms">Voltar</Link>
        </Button>
      </div>

      {responses.isLoading && <p className="text-sm text-muted-foreground">Carregando...</p>}
      {responses.error && <p className="text-sm text-destructive">{errorMessage(responses.error)}</p>}

      {responses.data?.responses.length === 0 && (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            Nenhuma resposta ainda. Compartilhe a URL pública do formulário.
          </CardContent>
        </Card>
      )}

      <div className="space-y-3">
        {responses.data?.responses.map((response) => (
          <Card key={response.id}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                #{response.id} — {new Date(response.createdAt).toLocaleString("pt-BR")}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <dl className="grid gap-2 sm:grid-cols-2">
                {response.answers.map((answer, i) => (
                  <div key={i}>
                    <dt className="text-xs text-muted-foreground">{answer.label}</dt>
                    <dd className="text-sm">{formatValue(answer)}</dd>
                  </div>
                ))}
              </dl>
            </CardContent>
          </Card>
        ))}
      </div>
    </AppShell>
  );
}

// O backend guarda tudo como texto mais o tipo; a formatação é da tela.
function formatValue(answer: { type: string; value: string }) {
  if (answer.type === "checkbox") return answer.value === "true" ? "Sim" : "Não";
  return answer.value === "" ? "—" : answer.value;
}
