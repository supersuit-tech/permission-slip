/**
 * Bulk approval group review — per-item approve/deny toggles defaulting to approve.
 */
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { useApprovalBulkGroup } from "../../hooks/useApprovalBulkGroup";
import { useBulkDecideApprovalGroup } from "../../hooks/useBulkDecideApprovalGroup";
import { useActionSchema } from "../../hooks/useActionSchema";
import { colors } from "../../theme/colors";
import { buildActionSummary } from "./approvalUtils";

type Props = NativeStackScreenProps<RootStackParamList, "BulkApprovalGroup">;

type ItemDecision = "approve" | "deny";

export default function BulkApprovalGroupScreen({ route, navigation }: Props) {
  const { bulkGroupId } = route.params;
  const { data: group, isLoading, error } = useApprovalBulkGroup(bulkGroupId);
  const { displayTemplate } = useActionSchema(group?.action_type ?? "");
  const decide = useBulkDecideApprovalGroup();
  const [decisions, setDecisions] = useState<Record<string, ItemDecision>>({});

  useEffect(() => {
    if (!group?.items) return;
    setDecisions((prev) => {
      const next = { ...prev };
      for (const item of group.items) {
        if (!(item.approval_id in next)) next[item.approval_id] = "approve";
      }
      return next;
    });
  }, [group?.items]);

  const pendingItems = useMemo(
    () => group?.items.filter((i) => i.status === "pending") ?? [],
    [group?.items],
  );

  const setAll = useCallback(
    (decision: ItemDecision) => {
      setDecisions((prev) => {
        const next = { ...prev };
        for (const item of pendingItems) next[item.approval_id] = decision;
        return next;
      });
    },
    [pendingItems],
  );

  const onSubmit = async () => {
    if (!group) return;
    await decide.mutateAsync({
      bulkGroupId: group.bulk_group_id,
      decisions: pendingItems.map((item) => ({
        approval_id: item.approval_id,
        decision: decisions[item.approval_id] ?? "approve",
      })),
    });
    navigation.goBack();
  };

  if (isLoading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color={colors.primary} />
      </View>
    );
  }

  if (error || !group) {
    return (
      <View style={styles.center}>
        <Text>Could not load bulk group.</Text>
      </View>
    );
  }

  return (
    <ScrollView contentContainerStyle={styles.container}>
      <Text style={styles.title}>
        {group.action_type} ({group.item_count} items)
      </Text>
      <View style={styles.row}>
        <TouchableOpacity style={styles.chip} onPress={() => setAll("approve")}>
          <Text>Approve all</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.chip} onPress={() => setAll("deny")}>
          <Text>Deny all</Text>
        </TouchableOpacity>
      </View>
      {group.items.map((item) => (
        <View key={item.approval_id} style={styles.card}>
          <Text style={styles.summary}>
            {buildActionSummary(
              item.action.type,
              item.action.parameters as Record<string, unknown>,
              displayTemplate,
              item.resource_details as Record<string, unknown> | undefined,
            )}
          </Text>
          {item.status === "pending" && (
            <View style={styles.switchRow}>
              <Text>Approve</Text>
              <Switch
                value={(decisions[item.approval_id] ?? "approve") === "approve"}
                onValueChange={(v) =>
                  setDecisions((prev) => ({
                    ...prev,
                    [item.approval_id]: v ? "approve" : "deny",
                  }))
                }
              />
            </View>
          )}
        </View>
      ))}
      <TouchableOpacity
        style={styles.submit}
        disabled={decide.isPending || pendingItems.length === 0}
        onPress={() => void onSubmit()}
      >
        <Text style={styles.submitText}>Submit review</Text>
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: "center", justifyContent: "center" },
  container: { padding: 16, gap: 12 },
  title: { fontSize: 18, fontWeight: "600" },
  row: { flexDirection: "row", gap: 8 },
  chip: {
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 8,
    backgroundColor: colors.gray100,
  },
  card: {
    padding: 12,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.gray200,
    gap: 8,
  },
  summary: { fontSize: 14 },
  switchRow: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  submit: {
    marginTop: 8,
    backgroundColor: colors.primary,
    padding: 14,
    borderRadius: 8,
    alignItems: "center",
  },
  submitText: { color: "#fff", fontWeight: "600" },
});
