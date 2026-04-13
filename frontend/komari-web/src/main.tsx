import React, { StrictMode, useMemo } from "react";
import { createRoot } from "react-dom/client";
import "./global.css";
import { Theme } from "@radix-ui/themes";
import "@radix-ui/themes/styles.css";
import {
  ThemeContext,
  THEME_DEFAULTS,
  allowedAppearances,
  allowedColors,
  type Appearance,
  type Colors,
} from "./contexts/ThemeContext";
import { useLocalStorage } from "./hooks/useLocalStorage";
import { useSystemTheme } from "./hooks/useSystemTheme";
import { BrowserRouter } from "react-router-dom";
// Ensure i18n is initialized before any component renders
import "./i18n/config";
import ErrorBoundary from "./components/ErrorBoundary";
import { Suspense } from "react";
import { useRoutes } from "react-router-dom";
import { routes } from "./routes";
import Loading from "./components/loading";
import { PublicInfoProvider } from "./contexts/PublicInfoContext";
import { PWAInstallPrompt } from "./components/PWAInstallPrompt";
import { PWAUpdatePrompt } from "./components/PWAUpdatePrompt";
import { OfflineIndicator } from "./components/OfflineIndicator";
import { Toaster } from "./components/ui/sonner";
import { RPC2Provider } from "./contexts/RPC2Context";
import { AccountProvider, useAccount } from "./contexts/AccountContext";

const isAppearance = (value: unknown): value is Appearance =>
  typeof value === "string" &&
  (allowedAppearances as readonly string[]).includes(value);

const isColor = (value: unknown): value is Colors =>
  typeof value === "string" &&
  (allowedColors as readonly string[]).includes(value);

async function saveThemePreferences(
  appearance: Appearance,
  color: Colors,
): Promise<void> {
  const response = await fetch("/api/admin/account/theme", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      theme_appearance: appearance,
      theme_color: color,
    }),
  });

  if (!response.ok) {
    throw new Error(`Failed to save theme preferences: ${response.status}`);
  }
}

const App = () => {
  const { account, loading: accountLoading } = useAccount();
  React.useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const tempKey = params.get("temp_key");

    if (tempKey) {
      document.cookie = `temp_key=${tempKey}; path=/; max-age=${60 * 60 * 24 * 365 * 100}`;
      params.delete("temp_key");
      window.history.replaceState(
        {},
        document.title,
        `${window.location.pathname}${params.toString() ? "?" + params.toString() : ""}`,
      );
    }
  }, []);
  const [appearance, setAppearance] = useLocalStorage<Appearance>(
    "appearance",
    THEME_DEFAULTS.appearance,
  );
  const [color, setColor] = useLocalStorage<Colors>(
    "color",
    THEME_DEFAULTS.color,
  );
  const hydratedAccountRef = React.useRef<string | null>(null);
  const skipNextPersistRef = React.useRef(false);

  React.useEffect(() => {
    if (!isAppearance(appearance)) {
      setAppearance(THEME_DEFAULTS.appearance);
    }
  }, [appearance, setAppearance]);

  React.useEffect(() => {
    if (!isColor(color)) {
      setColor(THEME_DEFAULTS.color);
    }
  }, [color, setColor]);

  // Use the system theme hook to resolve "system" to actual theme
  const resolvedAppearance = useSystemTheme(appearance);

  React.useEffect(() => {
    if (accountLoading) {
      return;
    }

    if (!account?.logged_in || !account.uuid) {
      hydratedAccountRef.current = null;
      return;
    }

    if (hydratedAccountRef.current === account.uuid) {
      return;
    }

    const serverAppearance = isAppearance(account.theme_appearance)
      ? account.theme_appearance
      : null;
    const serverColor = isColor(account.theme_color) ? account.theme_color : null;

    if (serverAppearance || serverColor) {
      skipNextPersistRef.current = true;
      if (serverAppearance && serverAppearance !== appearance) {
        setAppearance(serverAppearance);
      }
      if (serverColor && serverColor !== color) {
        setColor(serverColor);
      }
    } else {
      void saveThemePreferences(appearance, color).catch((error) => {
        console.warn("Failed to seed account theme preferences:", error);
      });
    }

    hydratedAccountRef.current = account.uuid;
  }, [accountLoading, account?.logged_in, account?.uuid]);

  React.useEffect(() => {
    if (accountLoading || !account?.logged_in || !account.uuid) {
      return;
    }

    if (hydratedAccountRef.current !== account.uuid) {
      return;
    }

    if (skipNextPersistRef.current) {
      skipNextPersistRef.current = false;
      return;
    }

    const timer = window.setTimeout(() => {
      void saveThemePreferences(appearance, color).catch((error) => {
        console.warn("Failed to sync theme preferences:", error);
      });
    }, 300);

    return () => {
      window.clearTimeout(timer);
    };
  }, [appearance, color, accountLoading, account?.logged_in, account?.uuid]);

  const themeContextValue = useMemo(
    () => ({
      appearance,
      setAppearance,
      color,
      setColor,
    }),
    [appearance, setAppearance, color, setColor],
  );
  const routing = useRoutes(routes);
  return (
    <Suspense fallback={<Loading />}>
      <ThemeContext.Provider value={themeContextValue}>
        <Theme
          appearance={resolvedAppearance}
          accentColor={color}
          scaling="110%"
          className="theme-root"
          style={{
            backgroundColor: "transparent",
            minHeight: "100vh",
          }}
        >
          <RPC2Provider>
            <PublicInfoProvider>
              <Toaster />
              <OfflineIndicator />
              {routing}
              <PWAInstallPrompt />
              <PWAUpdatePrompt />
            </PublicInfoProvider>
          </RPC2Provider>
        </Theme>
      </ThemeContext.Provider>
    </Suspense>
  );
};

createRoot(document.getElementById("root")!).render(
  <ErrorBoundary>
    <StrictMode>
      <BrowserRouter>
        <AccountProvider>
          <App />
        </AccountProvider>
      </BrowserRouter>
    </StrictMode>
  </ErrorBoundary>,
);
