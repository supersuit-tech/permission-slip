import { useEffect } from "react";
import { ActivityIndicator, StyleSheet, Text, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { RootStackParamList } from "../../navigation/RootNavigator";
import { colors } from "../../theme/colors";

type Props = NativeStackScreenProps<RootStackParamList, "DeepLinkBulkGroup">;

export default function DeepLinkBulkGroupScreen({ route, navigation }: Props) {
  const { bulkGroupId } = route.params;

  useEffect(() => {
    navigation.replace("BulkApprovalGroup", { bulkGroupId });
  }, [bulkGroupId, navigation]);

  return (
    <View style={styles.container}>
      <ActivityIndicator size="large" color={colors.primary} />
      <Text style={styles.text}>Loading bulk approval…</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.white,
  },
  text: { marginTop: 12, fontSize: 14, color: colors.gray500 },
});
