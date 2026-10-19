import { useState } from "react";
import { useParams } from "@tanstack/react-router";
import { useMutation, useQuery } from "@tanstack/react-query";
import { api, errorMessage, fieldErrors } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

// Página pública: não usa token nenhum. A validação de verdade é a do
// backend; aqui só mostramos os erros que ele devolve, por campo.
export function PublicFormPage() {
  const { slug } = useParams({ from: "/f/$slug" });
  const [answers, setAnswers] = useState<Record<string, string | boolean>>({});

  const form = useQuery({
    queryKey: ["public", slug],
    queryFn: async () => {
      const { data, error } = await api.GET("/public/forms/{slug}", {
        params: { path: { slug } },
      });
      if (error) throw error;
      return data;
    },
  });

  const submit = useMutation({
    mutationFn: async () => {
      const { data, error } = await api.POST("/public/forms/{slug}/responses", {
        params: { path: { slug } },
        body: { answers },
      });
      if (error) throw error;
      return data;
    },
  });

  if (form.isLoading) {
    return <CenteredMessage>Carregando...</CenteredMessage>;
  }
  if (form.error) {
    return (
      <CenteredMessage>
        <span className="text-destructive">{errorMessage(form.error, "Formulário não encontrado")}</span>
      </CenteredMessage>
    );
  }
  if (submit.isSuccess) {
    return (
      <CenteredMessage>
        <p className="text-lg font-medium">Resposta enviada. Obrigado!</p>
      </CenteredMessage>
    );
  }

  const errors = fieldErrors(submit.error);

  return (
    <div className="min-h-screen bg-muted/30 px-4 py-10">
      <Card className="mx-auto max-w-xl">
        <CardHeader>
          <CardTitle>{form.data!.title}</CardTitle>
          {form.data!.description && <CardDescription>{form.data!.description}</CardDescription>}
        </CardHeader>
        <CardContent>
          <form
            className="space-y-5"
            onSubmit={(e) => {
              e.preventDefault();
              submit.mutate();
            }}
          >
            {form.data!.fields.map((field) => {
              const id = String(field.id);
              const error = errors[id];
              return (
                <div key={field.id} className="space-y-1.5">
                  {field.type !== "checkbox" && (
                    <Label htmlFor={id}>
                      {field.label}
                      {field.required && <span className="ml-1 text-destructive">*</span>}
                    </Label>
                  )}

                  {field.type === "text" && (
                    <Input
                      id={id}
                      value={(answers[id] as string) ?? ""}
                      onChange={(e) => setAnswers({ ...answers, [id]: e.target.value })}
                    />
                  )}

                  {field.type === "number" && (
                    <Input
                      id={id}
                      type="number"
                      value={(answers[id] as string) ?? ""}
                      onChange={(e) => setAnswers({ ...answers, [id]: e.target.value })}
                    />
                  )}

                  {field.type === "select" && (
                    <Select
                      value={(answers[id] as string) ?? ""}
                      onValueChange={(value) => setAnswers({ ...answers, [id]: value })}
                    >
                      <SelectTrigger id={id}>
                        <SelectValue placeholder="Selecione..." />
                      </SelectTrigger>
                      <SelectContent>
                        {(field.options ?? []).map((option) => (
                          <SelectItem key={option} value={option}>
                            {option}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}

                  {field.type === "checkbox" && (
                    <div className="flex items-center gap-2">
                      <Checkbox
                        id={id}
                        checked={answers[id] === true}
                        onCheckedChange={(checked) => setAnswers({ ...answers, [id]: checked === true })}
                      />
                      <Label htmlFor={id}>
                        {field.label}
                        {field.required && <span className="ml-1 text-destructive">*</span>}
                      </Label>
                    </div>
                  )}

                  {error && <p className="text-sm text-destructive">{error}</p>}
                </div>
              );
            })}

            {submit.isError && Object.keys(errors).length === 0 && (
              <p className="text-sm text-destructive">{errorMessage(submit.error)}</p>
            )}

            <Button type="submit" disabled={submit.isPending}>
              {submit.isPending ? "Enviando..." : "Enviar resposta"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

function CenteredMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center px-4 text-center text-sm">{children}</div>
  );
}
