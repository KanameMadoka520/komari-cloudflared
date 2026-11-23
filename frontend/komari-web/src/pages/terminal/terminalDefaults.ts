import {
  readCurrentThemeSettings,
  saveThemeSettings,
} from "@/utils/saveThemeSettings";

export const MIN_TERMINAL_FONT_SIZE = 10;

export const DEFAULT_TERMINAL_BACKGROUND_URL = "";
export const DEFAULT_TERMINAL_BACKGROUND_OFFSET_X = 0;
export const DEFAULT_TERMINAL_BACKGROUND_OFFSET_Y = 0;
export const MIN_TERMINAL_BACKGROUND_OFFSET = -100;
export const MAX_TERMINAL_BACKGROUND_OFFSET = 100;
export const DEFAULT_TERMINAL_BACKGROUND_SCALE = 100;
export const MIN_TERMINAL_BACKGROUND_SCALE = 10;
export const MAX_TERMINAL_BACKGROUND_SCALE = 300;
export const DEFAULT_TERMINAL_BACKGROUND_OPACITY = 55;
export const MIN_TERMINAL_BACKGROUND_OPACITY = 0;
export const MAX_TERMINAL_BACKGROUND_OPACITY = 100;
export const DEFAULT_TERMINAL_BACKGROUND_BLUR = 0;
export const MIN_TERMINAL_BACKGROUND_BLUR = 0;
export const MAX_TERMINAL_BACKGROUND_BLUR = 40;

export const TERMINAL_SETTINGS_KEY = "_komari_terminal_settings_v1";

export interface TerminalSettings {
  backgroundUrl: string;
  backgroundOffsetX: number;
  backgroundOffsetY: number;
  backgroundScale: number;
  backgroundOpacity: number;
  backgroundBlur: number;
}

export const DEFAULT_TERMINAL_SETTINGS: TerminalSettings = {
  backgroundUrl: DEFAULT_TERMINAL_BACKGROUND_URL,
  backgroundOffsetX: DEFAULT_TERMINAL_BACKGROUND_OFFSET_X,
  backgroundOffsetY: DEFAULT_TERMINAL_BACKGROUND_OFFSET_Y,
  backgroundScale: DEFAULT_TERMINAL_BACKGROUND_SCALE,
  backgroundOpacity: DEFAULT_TERMINAL_BACKGROUND_OPACITY,
  backgroundBlur: DEFAULT_TERMINAL_BACKGROUND_BLUR,
};

export const clampTerminalBackgroundOffset = (value: number): number => {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.min(
    MAX_TERMINAL_BACKGROUND_OFFSET,
    Math.max(MIN_TERMINAL_BACKGROUND_OFFSET, Math.round(value)),
  );
};

export const clampTerminalBackgroundScale = (value: number): number => {
  if (!Number.isFinite(value)) {
    return DEFAULT_TERMINAL_BACKGROUND_SCALE;
  }
  return Math.min(
    MAX_TERMINAL_BACKGROUND_SCALE,
    Math.max(MIN_TERMINAL_BACKGROUND_SCALE, Math.round(value)),
  );
};

export const clampTerminalBackgroundOpacity = (value: number): number => {
  if (!Number.isFinite(value)) {
    return DEFAULT_TERMINAL_BACKGROUND_OPACITY;
  }
  return Math.min(
    MAX_TERMINAL_BACKGROUND_OPACITY,
    Math.max(MIN_TERMINAL_BACKGROUND_OPACITY, Math.round(value)),
  );
};

export const clampTerminalBackgroundBlur = (value: number): number => {
  if (!Number.isFinite(value)) {
    return DEFAULT_TERMINAL_BACKGROUND_BLUR;
  }
  return Math.min(
    MAX_TERMINAL_BACKGROUND_BLUR,
    Math.max(MIN_TERMINAL_BACKGROUND_BLUR, Math.round(value)),
  );
};

export const isLikelyVideoUrl = (url: string): boolean => {
  const trimmed = url.trim();
  if (!trimmed) return false;
  if (/^data:video\//i.test(trimmed)) return true;
  return /\.(mp4|webm|ogg|ogv|mov|m4v)(\?.*)?(#.*)?$/i.test(trimmed);
};

export const parseTerminalSettings = (value: unknown): TerminalSettings => {
  if (!value) {
    return { ...DEFAULT_TERMINAL_SETTINGS };
  }
  let record: Partial<TerminalSettings> | null = null;
  if (typeof value === "string") {
    try {
      const parsed = JSON.parse(value);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        record = parsed as Partial<TerminalSettings>;
      }
    } catch {
      return { ...DEFAULT_TERMINAL_SETTINGS };
    }
  } else if (typeof value === "object" && !Array.isArray(value)) {
    record = value as Partial<TerminalSettings>;
  }
  if (!record) {
    return { ...DEFAULT_TERMINAL_SETTINGS };
  }
  const backgroundUrl =
    typeof record.backgroundUrl === "string"
      ? record.backgroundUrl.trim()
      : DEFAULT_TERMINAL_BACKGROUND_URL;
  const backgroundOffsetX =
    typeof record.backgroundOffsetX === "number"
      ? clampTerminalBackgroundOffset(record.backgroundOffsetX)
      : DEFAULT_TERMINAL_BACKGROUND_OFFSET_X;
  const backgroundOffsetY =
    typeof record.backgroundOffsetY === "number"
      ? clampTerminalBackgroundOffset(record.backgroundOffsetY)
      : DEFAULT_TERMINAL_BACKGROUND_OFFSET_Y;
  const backgroundScale =
    typeof record.backgroundScale === "number"
      ? clampTerminalBackgroundScale(record.backgroundScale)
      : DEFAULT_TERMINAL_BACKGROUND_SCALE;
  const backgroundOpacity =
    typeof record.backgroundOpacity === "number"
      ? clampTerminalBackgroundOpacity(record.backgroundOpacity)
      : DEFAULT_TERMINAL_BACKGROUND_OPACITY;
  const backgroundBlur =
    typeof record.backgroundBlur === "number"
      ? clampTerminalBackgroundBlur(record.backgroundBlur)
      : DEFAULT_TERMINAL_BACKGROUND_BLUR;
  return {
    backgroundUrl,
    backgroundOffsetX,
    backgroundOffsetY,
    backgroundScale,
    backgroundOpacity,
    backgroundBlur,
  };
};

let cachedTerminalSettings: TerminalSettings = { ...DEFAULT_TERMINAL_SETTINGS };
let cachedThemeName: string | null = null;

export const getCachedTerminalSettings = (): TerminalSettings => ({
  ...cachedTerminalSettings,
});

export const setCachedTerminalSettings = (settings: TerminalSettings): void => {
  cachedTerminalSettings = { ...settings };
};

export const fetchTerminalSettingsFromTheme =
  async (): Promise<TerminalSettings> => {
    const { theme, settings } = await readCurrentThemeSettings();
    cachedThemeName = theme;
    const parsed = parseTerminalSettings(settings[TERMINAL_SETTINGS_KEY]);
    cachedTerminalSettings = { ...parsed };
    return parsed;
  };

export const persistTerminalSettingsToTheme = async (
  settings: TerminalSettings,
): Promise<void> => {
  cachedTerminalSettings = { ...settings };
  const theme = cachedThemeName ?? (await readCurrentThemeSettings()).theme;
  cachedThemeName = theme;
  await saveThemeSettings(theme, {
    [TERMINAL_SETTINGS_KEY]: settings,
  });
};
