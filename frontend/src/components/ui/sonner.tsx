import { Toaster as Sonner } from "sonner";

function Toaster(props: React.ComponentProps<typeof Sonner>) {
  return <Sonner richColors {...props} />;
}

export { Toaster };
