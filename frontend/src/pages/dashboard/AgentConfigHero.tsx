import { Link } from "react-router-dom";
import { Plug, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

interface ConfigStepProps {
  step: number;
  icon: React.ReactNode;
  title: string;
  description: string;
}

function ConfigStep({ step, icon, title, description }: ConfigStepProps) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide">
        Step {step}
      </div>
      <div className="bg-primary/10 text-primary mb-3 flex size-12 items-center justify-center rounded-full">
        {icon}
      </div>
      <h3 className="mb-1 text-sm font-semibold">{title}</h3>
      <p className="text-muted-foreground max-w-[200px] text-xs">
        {description}
      </p>
    </div>
  );
}

interface AgentConfigHeroProps {
  agentId: number;
}

export function AgentConfigHero({ agentId }: AgentConfigHeroProps) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center px-6 py-16 text-center md:py-24">
        <div className="bg-primary/10 text-primary mb-6 flex size-16 items-center justify-center rounded-full">
          <Plug className="size-8" />
        </div>

        <h1 className="mb-3 text-2xl font-bold tracking-tight">
          Your agent is ready &mdash; now give it superpowers
        </h1>
        <p className="text-muted-foreground mb-12 max-w-md text-sm">
          Connect services like GitHub, Gmail, or Slack so your agent can take
          actions on your behalf. You&rsquo;ll approve every action before it
          happens.
        </p>

        <div className="mb-12 grid w-full max-w-lg grid-cols-1 gap-8 md:grid-cols-2">
          <ConfigStep
            step={1}
            icon={<Plug className="size-6" />}
            title="Add a connector"
            description="Choose which services your agent can interact with"
          />
          <ConfigStep
            step={2}
            icon={<ShieldCheck className="size-6" />}
            title="Set permissions"
            description="Approve, deny, or create standing rules for every action"
          />
        </div>

        <Button size="lg" asChild>
          <Link to={`/agents/${agentId}`}>Configure Your Agent</Link>
        </Button>
      </CardContent>
    </Card>
  );
}
