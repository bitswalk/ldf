import type { Component } from "solid-js";
import { createSignal, Match, onMount, Switch } from "solid-js";
import { Transition } from "solid-transition-group";
import { Header } from "./components/Header";
import { Distribution } from "./views/Distribution";
import { DistributionDetail } from "./views/DistributionDetail";
import { Artifacts } from "./views/Artifacts";
import { Sources } from "./views/Sources";
import { SourceDetails } from "./views/Sources/SourceDetails";
import { Components, ComponentDetails } from "./views/Components";
import { Login } from "./views/Login";
import { Register } from "./views/Register";
import { Connection } from "./views/Connection";
import { Settings } from "./views/Settings";
import { BoardProfiles } from "./views/BoardProfiles";
import { Console } from "./components/Console";
import { Menu, type MenuItem } from "./components/Menu";
import { logout, ensureValidToken, type UserInfo } from "./services/auth";
import { onAuthChange } from "./services/api";
import type { APIInfo } from "./services/storage";
import {
  getServerUrl,
  setServerUrl,
  getAuthToken,
  getUserInfo,
  setAuthToken,
  setUserInfo,
  setRefreshToken,
  setTokenExpiresAt,
  setAPIEndpoints,
  hasServerConnection,
  hasCompleteServerConnection,
  hasAuthSession,
  discoverAPIEndpoints,
  clearServerUrl,
  clearAllAuth,
} from "./services/storage";
import { syncDevModeFromServer } from "./services/settings";
import { initializeFavicon, initializeAppName } from "./services/branding";

type ViewType =
  | "server-connection"
  | "distribution"
  | "distribution-detail"
  | "artifacts"
  | "sources"
  | "source-details"
  | "components"
  | "component-details"
  | "board-profiles"
  | "login"
  | "register"
  | "settings";

interface AuthState {
  serverUrl: string;
  user: UserInfo | null;
  token: string | null;
}

const App: Component = () => {
  const [currentView, setCurrentView] =
    createSignal<ViewType>("server-connection");
  const [isLoggedIn, setIsLoggedIn] = createSignal(false);
  const [authState, setAuthState] = createSignal<AuthState>({
    serverUrl: "",
    user: null,
    token: null,
  });
  const [pendingUsername, setPendingUsername] = createSignal("");
  const [connectionError, setConnectionError] = createSignal<string | null>(
    null,
  );
  const [selectedDistributionId, setSelectedDistributionId] = createSignal<
    string | null
  >(null);
  const [selectedSourceId, setSelectedSourceId] = createSignal<string | null>(
    null,
  );
  const [sourceDetailsReturnView, setSourceDetailsReturnView] = createSignal<
    "sources" | "settings"
  >("sources");
  const [selectedComponentId, setSelectedComponentId] = createSignal<
    string | null
  >(null);

  onMount(async () => {
    // Subscribe to auth changes from the API client
    // This handles cases where token refresh fails during API calls
    const unsubscribe = onAuthChange((isAuthenticated) => {
      if (!isAuthenticated && isLoggedIn()) {
        // Auth was invalidated (e.g., refresh token expired)
        setAuthState((prev) => ({ ...prev, user: null, token: null }));
        setIsLoggedIn(false);
        setCurrentView("login");
      }
    });

    // If we have a server URL but missing endpoints, try to re-discover them
    if (hasServerConnection() && !hasCompleteServerConnection()) {
      const result = await discoverAPIEndpoints();
      if (!result.success) {
        // Discovery failed - clear the invalid URL and show error
        setConnectionError(
          `Failed to connect to saved server: ${result.error}`,
        );
        clearServerUrl();
        setCurrentView("server-connection");
        return;
      }
    }

    if (hasAuthSession() && hasCompleteServerConnection()) {
      const serverUrl = getServerUrl()!;
      const token = getAuthToken()!;
      const user = getUserInfo()!;

      // Validate the stored token against the server
      const isValid = await ensureValidToken();
      if (isValid) {
        // Token is valid (or was successfully refreshed)
        setAuthState({ serverUrl, user, token: getAuthToken()! });
        setIsLoggedIn(true);
        setCurrentView("distribution");
        // Sync devmode setting from server for root users
        syncDevModeFromServer();
        // Initialize branding (favicon and app name)
        initializeFavicon();
        initializeAppName();
      } else {
        // Token is invalid and couldn't be refreshed - redirect to login
        clearAllAuth();
        setAuthState((prev) => ({ ...prev, serverUrl }));
        setCurrentView("login");
      }
    } else if (hasCompleteServerConnection()) {
      const serverUrl = getServerUrl()!;
      setAuthState((prev) => ({ ...prev, serverUrl }));
      setCurrentView("login");
    } else {
      setCurrentView("server-connection");
    }
  });

  const handleBadgeClick = () => {
    if (currentView() === "distribution") {
      if (isLoggedIn()) {
        setCurrentView("distribution");
      } else {
        setCurrentView("login");
      }
    } else {
      setCurrentView("distribution");
    }
  };

  const handleLogout = async () => {
    const result = await logout();

    // Clear local auth state regardless of server response
    // (user wants to log out, so we clear local state even if server fails)
    clearAllAuth();
    setAuthState((prev) => ({ ...prev, user: null, token: null }));
    setIsLoggedIn(false);
    setCurrentView("login");
  };

  const handleToggleLogin = () => {
    setIsLoggedIn(!isLoggedIn());
  };

  const handleServerConnect = (serverUrl: string, apiInfo: APIInfo) => {
    setServerUrl(serverUrl);
    setAPIEndpoints(apiInfo.endpoints);
    setAuthState((prev) => ({ ...prev, serverUrl }));
    setCurrentView("login");
  };

  const handleLoginSuccess = (
    serverUrl: string,
    user: UserInfo,
    token: string,
    refreshToken: string,
    expiresAt: string,
  ) => {
    setServerUrl(serverUrl);
    setAuthToken(token);
    setRefreshToken(refreshToken);
    setTokenExpiresAt(expiresAt);
    setUserInfo(user);
    setAuthState({ serverUrl, user, token });
    setIsLoggedIn(true);
    setCurrentView("distribution");
    // Sync devmode setting from server for root users
    syncDevModeFromServer();
    // Initialize branding (favicon and app name)
    initializeFavicon();
    initializeAppName();
  };

  const handleShowRegister = (username: string) => {
    setPendingUsername(username);
    setCurrentView("register");
  };

  const handleRegisterSuccess = (
    user: UserInfo,
    token: string,
    refreshToken: string,
    expiresAt: string,
  ) => {
    const serverUrl = authState().serverUrl;
    setAuthToken(token);
    setRefreshToken(refreshToken);
    setTokenExpiresAt(expiresAt);
    setUserInfo(user);
    setAuthState({ serverUrl, user, token });
    setIsLoggedIn(true);
    setCurrentView("distribution");
  };

  const handleBackToLogin = () => {
    setCurrentView("login");
  };

  const handleOpenSettings = () => {
    setCurrentView("settings");
  };

  const handleBackFromSettings = () => {
    setCurrentView("distribution");
  };

  const handleViewDistribution = (distributionId: string) => {
    setSelectedDistributionId(distributionId);
    setCurrentView("distribution-detail");
  };

  const handleBackFromDistributionDetail = () => {
    setSelectedDistributionId(null);
    setCurrentView("distribution");
  };

  const handleViewSource = (
    sourceId: string,
    returnTo: "sources" | "settings" = "sources",
  ) => {
    setSelectedSourceId(sourceId);
    setSourceDetailsReturnView(returnTo);
    setCurrentView("source-details");
  };

  const handleBackFromSourceDetails = () => {
    setSelectedSourceId(null);
    setCurrentView(sourceDetailsReturnView());
  };

  const handleViewComponent = (componentId: string) => {
    setSelectedComponentId(componentId);
    setCurrentView("component-details");
  };

  const handleBackFromComponentDetails = () => {
    setSelectedComponentId(null);
    setCurrentView("components");
  };

  const menuItems = (): MenuItem[] => [
    {
      id: "distribution",
      label: "Distributions",
      icon: "linux-logo",
      onClick: () => setCurrentView("distribution"),
    },
    {
      id: "components",
      label: "Components",
      icon: "cube",
      onClick: () => setCurrentView("components"),
    },
    {
      id: "artifacts",
      label: "Artifacts",
      icon: "package",
      onClick: () => setCurrentView("artifacts"),
    },
    {
      id: "sources",
      label: "Sources",
      icon: "git-branch",
      onClick: () => setCurrentView("sources"),
    },
    {
      id: "board-profiles",
      label: "Board Profiles",
      icon: "cpu",
      onClick: () => setCurrentView("board-profiles"),
    },
  ];

  return (
    <>
      {/* Row 1: Header */}
      <header id="header" class="h-12 w-full shrink-0">
        <Header
          isLoggedIn={isLoggedIn()}
          user={authState().user}
          onLogout={handleLogout}
          onSettings={handleOpenSettings}
          onBadgeClick={handleBadgeClick}
        />
      </header>

      {/* Row 2: Menu + Content Area */}
      <section class="flex flex-1 overflow-hidden">
        <Menu
          orientation="vertical"
          items={menuItems()}
          activeItemId={currentView()}
        />

        <section class="flex flex-col flex-1 overflow-hidden">
          <main id="viewport" class="flex-1 relative overflow-auto">
            <Transition
              mode="outin"
              enterActiveClass="transition-opacity duration-300 ease-in"
              enterClass="opacity-0"
              enterToClass="opacity-100"
              exitActiveClass="transition-opacity duration-300 ease-in"
              exitClass="opacity-100"
              exitToClass="opacity-0"
            >
              <Switch>
                <Match when={currentView() === "server-connection"}>
                  <Connection
                    onConnect={handleServerConnect}
                    initialError={connectionError()}
                  />
                </Match>
                <Match when={currentView() === "distribution"}>
                  <Distribution
                    isLoggedIn={isLoggedIn()}
                    user={authState().user}
                    onViewDistribution={handleViewDistribution}
                  />
                </Match>
                <Match
                  when={
                    currentView() === "distribution-detail" &&
                    selectedDistributionId()
                  }
                >
                  <DistributionDetail
                    distributionId={selectedDistributionId()!}
                    onBack={handleBackFromDistributionDetail}
                    user={authState().user}
                  />
                </Match>
                <Match when={currentView() === "components"}>
                  <Components
                    isLoggedIn={isLoggedIn()}
                    user={authState().user}
                    onViewComponent={handleViewComponent}
                  />
                </Match>
                <Match
                  when={
                    currentView() === "component-details" &&
                    selectedComponentId()
                  }
                >
                  <ComponentDetails
                    componentId={selectedComponentId()!}
                    onBack={handleBackFromComponentDetails}
                    onDeleted={handleBackFromComponentDetails}
                    user={authState().user}
                  />
                </Match>
                <Match when={currentView() === "board-profiles"}>
                  <BoardProfiles
                    isLoggedIn={isLoggedIn()}
                    user={authState().user}
                  />
                </Match>
                <Match when={currentView() === "artifacts"}>
                  <Artifacts
                    isLoggedIn={isLoggedIn()}
                    user={authState().user}
                  />
                </Match>
                <Match when={currentView() === "sources"}>
                  <Sources
                    isLoggedIn={isLoggedIn()}
                    user={authState().user}
                    onViewSource={(id) => handleViewSource(id)}
                  />
                </Match>
                <Match
                  when={
                    currentView() === "source-details" && selectedSourceId()
                  }
                >
                  <SourceDetails
                    sourceId={selectedSourceId()!}
                    onBack={handleBackFromSourceDetails}
                    onDeleted={handleBackFromSourceDetails}
                    user={authState().user}
                  />
                </Match>
                <Match when={currentView() === "login"}>
                  <Login
                    serverUrl={authState().serverUrl}
                    onLoginSuccess={handleLoginSuccess}
                    onShowRegister={handleShowRegister}
                  />
                </Match>
                <Match when={currentView() === "register"}>
                  <Register
                    serverUrl={authState().serverUrl}
                    prefillUsername={pendingUsername()}
                    onSuccess={handleRegisterSuccess}
                    onBackToLogin={handleBackToLogin}
                  />
                </Match>
                <Match when={currentView() === "settings"}>
                  <Settings onBack={handleBackFromSettings} />
                </Match>
              </Switch>
            </Transition>
          </main>

          <Console
            isLoggedIn={isLoggedIn()}
            onToggleLogin={handleToggleLogin}
            currentView={currentView()}
            onViewChange={setCurrentView}
          />
        </section>
      </section>
    </>
  );
};

export default App;
