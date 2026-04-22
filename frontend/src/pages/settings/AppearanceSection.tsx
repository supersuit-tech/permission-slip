import { Palette } from "lucide-react";
import { useTheme, type ThemePreference } from "@/components/ThemeContext";
import { SegmentedControl } from "@/components/ui/segmented-control";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const OPTIONS: { label: string; value: ThemePreference }[] = [
  { label: "Light", value: "light" },
  { label: "Dark", value: "dark" },
  { label: "System", value: "system" },
];

export function AppearanceSection() {
  const { preference, setPreference } = useTheme();

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Palette className="text-muted-foreground size-5" />
          <CardTitle>Appearance</CardTitle>
        </div>
        <CardDescription>
          Choose how Permission Slip looks. &ldquo;System&rdquo; matches your
          device&rsquo;s current theme.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-2">
          <p className="text-sm font-medium">Theme</p>
          <SegmentedControl
            options={OPTIONS}
            value={preference}
            onChange={setPreference}
            ariaLabel="Theme"
          />
        </div>
      </CardContent>
    </Card>
  );
}
