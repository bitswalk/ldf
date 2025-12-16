import type { Component } from "solid-js";
import { createSignal, Match, onMount, Switch } from "solid-js";
import { Transition } from "solid-transition-group";
import { Header } from "./components/Header";
import { Distribution } from "./views/Distribution";
import { Login } from "./views/Login";
import { Register } from "./views/Register";
import { ServerConnection } from "./views/ServerConnection";
import { UserSettings } from "./views/UserSettings";
import { Console } from "./components/Console";
import { logout, type UserInfo } from "./services/authService";
import type { APIInfo } from "./services/storageService";
import {
  getServerUrl,
  setServerUrl,
  getAuthToken,
  getUserInfo,
  setAuthToken,
  setUserInfo,
  setAPIEndpoints,
  hasServerConnection,
  hasCompleteServerConnection,
  hasAuthSession,
  discoverAPIEndpoints,
  clearServerUrl,
  clearAllAuth,
} from "./services/storageService";

type ViewType =
  | "server-connection"
  | "distribution"
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

  onMount(async () => {
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
      setAuthState({ serverUrl, user, token });
      setIsLoggedIn(true);
      setCurrentView("distribution");
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
  ) => {
    setServerUrl(serverUrl);
    setAuthToken(token);
    setUserInfo(user);
    setAuthState({ serverUrl, user, token });
    setIsLoggedIn(true);
    setCurrentView("distribution");
  };

  const handleShowRegister = (username: string) => {
    setPendingUsername(username);
    setCurrentView("register");
  };

  const handleRegisterSuccess = (user: UserInfo, token: string) => {
    const serverUrl = authState().serverUrl;
    setAuthToken(token);
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

  return (
    <>
      <section id="header" class="h-[10vh] w-full">
        <Header
          isLoggedIn={isLoggedIn()}
          user={authState().user}
          onLogout={handleLogout}
          onSettings={handleOpenSettings}
          onBadgeClick={handleBadgeClick}
        />
      </section>
      <main id="viewport" class="h-[90vh] w-full relative">
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
              <ServerConnection
                onConnect={handleServerConnect}
                initialError={connectionError()}
              />
            </Match>
            <Match when={currentView() === "distribution"}>
              <Distribution isLoggedIn={isLoggedIn()} user={authState().user} />
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
              <UserSettings onBack={handleBackFromSettings} />
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
    </>
  );
};

export default App;
