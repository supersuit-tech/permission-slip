export const META_NAMESPACE_KEY = "$meta";

/** Drive and Sheets actions that can share a Shared Drive workspace standing set. */
export const GOOGLE_SHARED_DRIVE_WORKSPACE_ACTION_TYPES = [
  "google.upload_drive_file",
  "google.create_drive_folder",
  "google.list_drive_files",
  "google.search_drive",
  "google.get_drive_file",
  "google.sheets_read_range",
  "google.sheets_write_range",
  "google.sheets_append_rows",
  "google.sheets_list_sheets",
] as const;

const WORKSPACE_ACTION_TYPE_SET = new Set<string>(
  GOOGLE_SHARED_DRIVE_WORKSPACE_ACTION_TYPES,
);

export function isGoogleSharedDriveWorkspaceAction(actionType: string): boolean {
  return WORKSPACE_ACTION_TYPE_SET.has(actionType);
}

export function sharedDriveIdFromResourceDetails(
  resourceDetails?: Record<string, unknown> | null,
): string | null {
  const driveId = resourceDetails?.drive_id;
  if (typeof driveId !== "string" || driveId.length === 0) {
    return null;
  }
  return driveId;
}

export function sharedDriveNameFromResourceDetails(
  resourceDetails?: Record<string, unknown> | null,
): string | null {
  const driveName = resourceDetails?.drive_name;
  if (typeof driveName !== "string" || driveName.trim().length === 0) {
    return null;
  }
  return driveName.trim();
}

export function isGoogleSharedDriveWorkspaceApproval(
  actionType: string,
  resourceDetails?: Record<string, unknown> | null,
): boolean {
  return (
    isGoogleSharedDriveWorkspaceAction(actionType) &&
    sharedDriveIdFromResourceDetails(resourceDetails) !== null
  );
}

export function sharedDriveWorkspaceCheckboxLabel(
  resourceDetails?: Record<string, unknown> | null,
): string {
  const driveName = sharedDriveNameFromResourceDetails(resourceDetails);
  if (driveName) {
    return `Auto-approve routine Drive and Sheets work inside ${driveName}`;
  }
  return "Auto-approve routine Drive and Sheets work inside this Shared Drive";
}

function workspaceConstraintKeys(actionType: string): string[] {
  switch (actionType) {
    case "google.create_drive_folder":
      return ["parent_id"];
    case "google.get_drive_file":
      return ["file_id"];
    case "google.sheets_read_range":
    case "google.sheets_write_range":
    case "google.sheets_append_rows":
      return ["spreadsheet_id", "range"];
    case "google.sheets_list_sheets":
      return ["spreadsheet_id"];
    default:
      return ["folder_id"];
  }
}

export function buildSharedDriveWorkspaceConstraints(
  actionType: string,
  driveId: string,
): Record<string, unknown> {
  const constraints: Record<string, unknown> = {};
  for (const key of workspaceConstraintKeys(actionType)) {
    constraints[key] = "*";
  }
  constraints[META_NAMESPACE_KEY] = { drive_id: driveId };
  return constraints;
}

export function sharedDriveWorkspaceScopeLabel(
  resourceDetails?: Record<string, unknown> | null,
): string {
  const driveName = sharedDriveNameFromResourceDetails(resourceDetails);
  if (driveName) {
    return `inside ${driveName}`;
  }
  return "inside this Shared Drive";
}

function humanizeGoogleActionType(actionType: string): string {
  const short = actionType.replace(/^google\./, "").replace(/_/g, " ");
  if (short.length === 0) {
    return actionType;
  }
  return short.charAt(0).toUpperCase() + short.slice(1);
}

export function sharedDriveWorkspaceStandingApprovalName(
  actionType: string,
  resourceDetails?: Record<string, unknown> | null,
): string {
  return `${humanizeGoogleActionType(actionType)} — ${sharedDriveWorkspaceScopeLabel(resourceDetails)}`;
}
