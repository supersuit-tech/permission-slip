import { Link } from "react-router-dom";
import { Check, Plug, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

interface ConfigStepProps {
  step: number;
  icon: React.ReactNode;
  title: string;
  description: string;
  completed?: boolean;
}

function ConfigStep({
  step,
  icon,
  title,
  description,
  completed,
}: ConfigStepProps) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide">
        Step {step}
      </div>
      <div
        className={
          completed
            ? "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 mb-3 flex size-12 items-center justify-center rounded-full"
            : "bg-primary/10 text-primary mb-3 flex size-12 items-center justify-center rounded-full"
        }
      >
        {completed ? <Check className="size-6" /> : icon}
      </div>
      <h3
        className={`mb-1 text-sm font-semibold ${completed ? "text-muted-foreground line-through" : ""}`}
      >
        {title}
      </h3>
      <p className="text-muted-foreground max-w-[200px] text-xs">
        {description}
      </p>
    </div>
  );
}

function StepConnector() {
  return (
    <div className="hidden items-center justify-center md:flex">
      <div className="bg-border h-px w-8" />
    </div>
  );
}

interface AgentConfigHeroProps {
  agentId: number;
  agentName?: string;
}

export function AgentConfigHero({ agentId, agentName }: AgentConfigHeroProps) {
  const displayName = agentName ?? "Your agent";

  return (
    <Card>
      <CardContent className="flex flex-col items-center px-6 py-16 text-center md:py-24">
        <div className="bg-primary/10 text-primary mb-6 flex size-16 items-center justify-center rounded-full">
          <Plug className="size-8" />
        </div>

        <h1 className="mb-3 text-2xl font-bold tracking-tight">
          {displayName} is ready &mdash; now give it superpowers
        </h1>
        <p className="text-muted-foreground mb-12 max-w-md text-sm">
          Connect services like GitHub, Gmail, or Slack so Openclaw can take
          actions on your behalf. You&rsquo;ll approve every action before it
          happens.
        </p>

        <div className="mb-12 grid w-full max-w-2xl grid-cols-1 items-start gap-6 md:grid-cols-5">
          <ConfigStep
            step={1}
            icon={<Check className="size-6" />}
            title="Register agent"
            description="Connected and verified"
            completed
          />
          <StepConnector />
          <ConfigStep
            step={2}
            icon={<Plug className="size-6" />}
            title="Add a connector"
            description="Choose which services Openclaw can interact with"
          />
          <StepConnector />
          <ConfigStep
            step={3}
            icon={<ShieldCheck className="size-6" />}
            title="Set permissions"
            description="Approve, deny, or create standing rules for every action"
          />
        </div>

        <Button size="lg" asChild>
          <Link to={`/agents/${agentId}`}>Configure {displayName}</Link>
        </Button>
      </CardContent>
    </Card>
  );
}
