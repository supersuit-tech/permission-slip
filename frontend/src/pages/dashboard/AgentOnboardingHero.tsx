import { Bot, ShieldCheck, Activity } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

interface OnboardingStepProps {
  step: number;
  icon: React.ReactNode;
  title: string;
  description: string;
}

function OnboardingStep({ step, icon, title, description }: OnboardingStepProps) {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="text-muted-foreground mb-2 text-xs font-medium uppercase tracking-wide">
        Step {step}
      </div>
      <div className="bg-primary/10 text-primary mb-3 flex size-12 items-center justify-center rounded-full">
        {icon}
      </div>
      <h3 className="mb-1 text-sm font-semibold">{title}</h3>
      <p className="text-muted-foreground max-w-[200px] text-xs">{description}</p>
    </div>
  );
}

interface AgentOnboardingHeroProps {
  onRegisterAgent: () => void;
}

export function AgentOnboardingHero({ onRegisterAgent }: AgentOnboardingHeroProps) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center px-6 py-16 text-center md:py-24">
        <div className="bg-primary/10 text-primary mb-6 flex size-16 items-center justify-center rounded-full">
          <Bot className="size-8" />
        </div>

        <h1 className="mb-3 text-2xl font-bold tracking-tight">
          Control what Openclaw can do
        </h1>
        <p className="text-muted-foreground mb-12 max-w-md text-sm">
          Permission Slip lets you approve, deny, and set standing rules for the
          actions Openclaw takes. Connect Openclaw to get started.
        </p>

        <div className="mb-12 grid w-full max-w-2xl grid-cols-1 gap-8 md:grid-cols-3">
          <OnboardingStep
            step={1}
            icon={<Bot className="size-6" />}
            title="Connect Openclaw"
            description="Link Openclaw with a simple invite code"
          />
          <OnboardingStep
            step={2}
            icon={<ShieldCheck className="size-6" />}
            title="Set permissions"
            description="Approve or deny actions, or create standing rules"
          />
          <OnboardingStep
            step={3}
            icon={<Activity className="size-6" />}
            title="Monitor activity"
            description="Track every action Openclaw takes in real time"
          />
        </div>

        <Button size="lg" onClick={onRegisterAgent}>
          Connect Openclaw
        </Button>
      </CardContent>
    </Card>
  );
}
