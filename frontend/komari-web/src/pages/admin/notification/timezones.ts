export const DEFAULT_NOTIFICATION_TIMEZONE = "Asia/Shanghai";

export const NOTIFICATION_TIMEZONES = [
  "Asia/Shanghai",
  "UTC+8",
  "Asia/Hong_Kong",
  "Asia/Taipei",
  "Asia/Singapore",
  "Asia/Tokyo",
  "Asia/Seoul",
  "UTC",
  "Europe/London",
  "Europe/Paris",
  "America/New_York",
  "America/Los_Angeles",
  "Australia/Sydney",
] as const;

export function timezoneMessageKey(zone: string) {
  return `notification.templates.timezones.${zone.replace(/\+/g, "plus").replace(/\//g, "_")}`;
}
