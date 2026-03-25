/**
 * Deep linking configuration for React Navigation.
 *
 * Handles two URL patterns:
 * 1. Custom scheme: permissionslip://approve/{approvalId}
 * 2. Universal links: https://app.permissionslip.dev/approve/{approvalId}
 *
 * Deep links navigate to the DeepLinkDetail screen which fetches the full
 * approval data by ID (since deep links only carry the approval ID, not the
 * full ApprovalSummary object that ApprovalDetail requires).
 */
import * as Linking from "expo-linking";
import type { LinkingOptions } from "@react-navigation/native";
import type { RootStackParamList } from "./RootNavigator";

const prefix = Linking.createURL("/");

export const linking: LinkingOptions<RootStackParamList> = {
  prefixes: [prefix, "permissionslip://", "https://app.permissionslip.dev"],
  config: {
    screens: {
      DeepLinkDetail: "approve/:approvalId",
      ApprovalList: "",
    },
  },
};
