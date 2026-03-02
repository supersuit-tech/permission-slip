import { StyleSheet, Text, TouchableOpacity, View } from "react-native";
import { colors } from "../../theme/colors";
import type { ApprovalStatus } from "../../hooks/useApprovals";

interface StatusFilterTabsProps {
  selected: ApprovalStatus;
  onSelect: (status: ApprovalStatus) => void;
}

const TABS: { key: ApprovalStatus; label: string }[] = [
  { key: "pending", label: "Pending" },
  { key: "approved", label: "Approved" },
  { key: "denied", label: "Denied" },
];

/** Horizontal tab bar for switching between pending, approved, and denied approvals. */
export default function StatusFilterTabs({ selected, onSelect }: StatusFilterTabsProps) {
  return (
    <View style={styles.container}>
      {TABS.map((tab) => {
        const isActive = tab.key === selected;
        return (
          <TouchableOpacity
            key={tab.key}
            testID={`tab-${tab.key}`}
            accessibilityRole="tab"
            accessibilityState={{ selected: isActive }}
            accessibilityLabel={`${tab.label} approvals`}
            style={[styles.tab, isActive && styles.tabActive]}
            onPress={() => onSelect(tab.key)}
          >
            <Text style={[styles.tabText, isActive && styles.tabTextActive]}>
              {tab.label}
            </Text>
          </TouchableOpacity>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flexDirection: "row",
    paddingHorizontal: 16,
    paddingVertical: 8,
    gap: 8,
  },
  tab: {
    flex: 1,
    paddingVertical: 8,
    borderRadius: 8,
    alignItems: "center",
    backgroundColor: colors.gray100,
  },
  tabActive: {
    backgroundColor: colors.gray900,
  },
  tabText: {
    fontSize: 14,
    fontWeight: "500",
    color: colors.gray500,
  },
  tabTextActive: {
    color: colors.white,
  },
});
