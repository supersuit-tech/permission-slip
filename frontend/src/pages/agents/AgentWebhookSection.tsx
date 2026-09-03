import { useState } from "react";
import {
  AlertTriangle,
  BellRing,
  Loader2,
  Trash2,
  Zap,
} from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  useAgentWebhook,
  useClearAgentWebhook,
  useSetAgentWebhook,
  useTestAgentWebhook,
} from "@/hooks/useAgentWebhook";

type WakeProvider = "openclaw" | "grokbot";

function isWakeProvider(value: string | undefined): value is WakeProvider {
  return value === "openclaw" || value === "grokbot";
}

function providerLabel(provider: WakeProvider): string {
  return provider === "grokbot" ? "Grok Bot" : "OpenClaw";
}

interface AgentWebhookSectionProps {
  agentId: number;
}

export function AgentWebhookSection({ agentId }: AgentWebhookSectionProps) {
  const { webhook, isLoading, error } = useAgentWebhook(agentId);
  const { setWebhook, isPending: saving } = useSetAgentWebhook();
  const { testWebhook, isPending: testing } = useTestAgentWebhook();
  const { clearWebhook, isPending: clearing } = useClearAgentWebhook();

  const [configureOpen, setConfigureOpen] = useState(false);
  const [clearOpen, setClearOpen] = useState(false);
  const [provider, setProvider] = useState<WakeProvider>("openclaw");
  const [url, setUrl] = useState("");
  const [token, setToken] = useState("");
  const [lastTestMessage, setLastTestMessage] = useState<string | null>(null);
  const [lastTestSuccess, setLastTestSuccess] = useState<boolean | null>(null);
  const [lastTestLatencyMs, setLastTestLatencyMs] = useState<number | null>(
    null,
  );

  const busy = saving || testing || clearing;
  const configured = webhook?.configured === true;
  const configuredProvider: WakeProvider = isWakeProvider(webhook?.provider)
    ? webhook.provider
    : "openclaw";
  const grokBot = provider === "grokbot";

  function openConfigureDialog() {
    setProvider(configuredProvider);
    setUrl(webhook?.webhook_url ?? "");
    setToken("");
    setConfigureOpen(true);
  }

  async function handleSave() {
    const trimmedUrl = url.trim();
    const trimmedToken = token.trim();
    if (!trimmedUrl || !trimmedToken) {
      toast.error("URL and token are both required.");
      return;
    }

    try {
      const result = await setWebhook({
        agentId,
        url: trimmedUrl,
        token: trimmedToken,
        provider,
      });
      setConfigureOpen(false);
      setToken("");
      toast.success("Webhook saved.");
      if (result?.test) {
        setLastTestSuccess(result.test.success ?? false);
        setLastTestMessage(result.test.message ?? null);
        setLastTestLatencyMs(result.test.latency_ms ?? null);
        if (!result.test.success) {
          toast.error(result.test.message ?? "Test wake failed.");
        }
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save webhook.",
      );
    }
  }

  async function handleTestWake() {
    try {
      const result = await testWebhook({ agentId });
      const test = result?.test;
      setLastTestSuccess(test?.success ?? false);
      setLastTestMessage(test?.message ?? null);
      setLastTestLatencyMs(test?.latency_ms ?? null);
      if (test?.success) {
        toast.success(
          test.latency_ms != null
            ? `Test wake delivered (${test.latency_ms} ms).`
            : "Test wake delivered.",
        );
      } else {
        toast.error(test?.message ?? "Test wake failed.");
      }
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Webhook test failed.",
      );
    }
  }

  async function handleClear() {
    try {
      await clearWebhook({ agentId });
      setClearOpen(false);
      setLastTestMessage(null);
      setLastTestSuccess(null);
      setLastTestLatencyMs(null);
      toast.success("Webhook cleared.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to clear webhook.",
      );
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BellRing className="size-5" />
            Push Wake Webhook
          </CardTitle>
          <p className="text-muted-foreground mt-1 text-sm">
            Register an OpenClaw gateway or Grok Bot webhook so the server can
            wake the agent when an approval is resolved.
          </p>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div
              className="flex items-center justify-center py-8"
              role="status"
              aria-label="Loading webhook configuration"
            >
              <Loader2
                className="text-muted-foreground size-6 animate-spin"
                aria-hidden="true"
              />
            </div>
          ) : error ? (
            <p className="text-destructive text-sm">{error}</p>
          ) : !configured ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <BellRing className="text-muted-foreground mb-3 size-10" />
              <p className="text-muted-foreground mb-3 max-w-md text-sm">
                No push wake webhook configured. Set an OpenClaw gateway hooks
                URL or a Grok Bot Cursor webhook to receive instant wakes when
                approvals resolve.
              </p>
              <Button variant="outline" size="sm" onClick={openConfigureDialog}>
                Configure webhook
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="rounded-lg border p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-sm font-medium">
                    {configuredProvider === "grokbot"
                      ? "Webhook URL"
                      : "Hooks URL"}
                  </span>
                  <Badge variant="secondary">
                    {providerLabel(configuredProvider)}
                  </Badge>
                  <Badge
                    variant="secondary"
                    className="bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400"
                  >
                    Configured
                  </Badge>
                </div>
                <p className="mt-2 break-all font-mono text-sm">
                  {webhook?.webhook_url}
                </p>
              </div>

              {webhook?.warning ? (
                <div
                  className="flex gap-2 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200"
                  role="alert"
                >
                  <AlertTriangle className="mt-0.5 size-4 shrink-0" />
                  <p>{webhook.warning}</p>
                </div>
              ) : null}

              {lastTestMessage ? (
                <div
                  className={`rounded-lg border p-3 text-sm ${
                    lastTestSuccess
                      ? "border-green-200 bg-green-50 text-green-900 dark:border-green-900/50 dark:bg-green-950/30 dark:text-green-200"
                      : "border-destructive/30 bg-destructive/5 text-destructive"
                  }`}
                  role="status"
                >
                  <p className="font-medium">
                    {lastTestSuccess ? "Test wake succeeded" : "Test wake failed"}
                  </p>
                  <p className="mt-1">{lastTestMessage}</p>
                  {lastTestSuccess && lastTestLatencyMs != null ? (
                    <p className="text-muted-foreground mt-1">
                      Latency: {lastTestLatencyMs} ms
                    </p>
                  ) : null}
                </div>
              ) : null}

              <div className="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleTestWake}
                  disabled={busy}
                >
                  {testing ? (
                    <Loader2 className="size-3 animate-spin" />
                  ) : (
                    <Zap className="size-3" />
                  )}
                  Test wake
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={openConfigureDialog}
                  disabled={busy}
                >
                  Update webhook
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setClearOpen(true)}
                  disabled={busy}
                >
                  <Trash2 className="size-3" />
                  Clear
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={configureOpen} onOpenChange={setConfigureOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {configured ? "Update push wake webhook" : "Configure push wake webhook"}
            </DialogTitle>
            <DialogDescription>
              {grokBot
                ? "Enter your Grok Bot Cursor automation webhook URL and Authorization header value. The URL must be https://api2.cursor.sh/automations/webhook/…. A test wake runs after saving."
                : "Enter your OpenClaw gateway hooks base URL and bearer token. The URL must resolve to a private network address (tailnet, LAN, or loopback). A test wake runs after saving."}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="webhook-provider">Provider</Label>
              <select
                id="webhook-provider"
                aria-label="Provider"
                className="border-input bg-background h-9 w-full rounded-md border px-2 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]"
                value={provider}
                onChange={(e) => {
                  if (isWakeProvider(e.target.value)) {
                    setProvider(e.target.value);
                  }
                }}
              >
                <option value="openclaw">OpenClaw</option>
                <option value="grokbot">Grok Bot</option>
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="webhook-url">
                {grokBot ? "Webhook URL" : "Hooks URL"}
              </Label>
              <Input
                id="webhook-url"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder={
                  grokBot
                    ? "https://api2.cursor.sh/automations/webhook/…"
                    : "http://100.x.x.x:18789/hooks"
                }
                autoComplete="off"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="webhook-token">
                {grokBot ? "Authorization header" : "Hooks token"}
              </Label>
              <Input
                id="webhook-token"
                type="password"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder={
                  grokBot
                    ? "Value from the Grok Bot webhook Authorization header"
                    : "Bearer token from OpenClaw config"
                }
                autoComplete="new-password"
              />
              <p className="text-muted-foreground text-xs">
                Token is write-only — enter it each time you save.
              </p>
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setConfigureOpen(false)}
              disabled={saving}
            >
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className="animate-spin" />}
              Save webhook
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={clearOpen} onOpenChange={setClearOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Clear push wake webhook</DialogTitle>
            <DialogDescription>
              This removes the registered webhook URL and token for this agent.
              Approval wakes will fall back to heartbeat sweep and watcher until
              you configure a webhook again.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="secondary"
              onClick={() => setClearOpen(false)}
              disabled={clearing}
            >
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleClear} disabled={clearing}>
              {clearing && <Loader2 className="animate-spin" />}
              Clear webhook
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
