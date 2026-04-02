import { Text, View } from "react-native";
import { colors } from "../theme/colors";

interface BrandBadgeProps {
  size?: number;
}

/** Purple rounded-square "P" brand mark matching the web's nav header badge. */
export function BrandBadge({ size = 32 }: BrandBadgeProps) {
  const radius = Math.round(size * 0.25);
  const fontSize = Math.round(size * 0.5);

  return (
    <View
      style={{
        width: size,
        height: size,
        borderRadius: radius,
        backgroundColor: colors.primary,
        alignItems: "center",
        justifyContent: "center",
      }}
      accessibilityElementsHidden={true}
      importantForAccessibility="no-hide-descendants"
    >
      <Text style={{ color: colors.white, fontSize, fontWeight: "700" }}>
        P
      </Text>
    </View>
  );
}
