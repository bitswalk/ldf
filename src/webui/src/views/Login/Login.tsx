import type { Component } from "solid-js";
import { createSignal, Show } from "solid-js";
import { Icon } from "../../components/Icon";
import { login } from "../../services/auth";
import type { UserInfo } from "../../services/auth";
import { t } from "../../services/i18n";

interface LoginProps {
  serverUrl: string;
  onLoginSuccess: (
    serverUrl: string,
    user: UserInfo,
    token: string,
    refreshToken: string,
    expiresAt: string,
  ) => void;
  onShowRegister: (username: string) => void;
}

export const Login: Component<LoginProps> = (props) => {
  const [username, setUsername] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError(null);
    setIsLoading(true);

    const result = await login(username(), password());

    setIsLoading(false);

    if (result.success) {
      props.onLoginSuccess(
        props.serverUrl,
        result.user,
        result.token,
        result.refreshToken,
        result.expiresAt,
      );
    } else {
      switch (result.error) {
        case "user_not_found":
          props.onShowRegister(username());
          break;
        case "network_error":
          setError(t("auth.login.errors.networkError"));
          break;
        case "internal_error":
          setError(t("auth.login.errors.serverError"));
          break;
        default:
          setError(result.message || t("auth.login.errors.generic"));
      }
    }
  };

  return (
    <section class="h-full flex flex-col items-center justify-center p-8">
      <h1 class="text-4xl font-bold mb-2">{t("auth.login.title")}</h1>
      <p class="text-muted-foreground mb-8 flex items-center gap-2">
        <Icon name="plugs" size="sm" />
        {props.serverUrl}
      </p>

      <form onSubmit={handleSubmit} class="w-full max-w-md flex flex-col gap-4">
        <Show when={error()}>
          <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
            {error()}
          </aside>
        </Show>

        <fieldset class="flex flex-col gap-4" disabled={isLoading()}>
          <legend class="sr-only">{t("auth.login.form.username.label")}</legend>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="user" size="sm" />
              {t("auth.login.form.username.label")}
            </span>
            <input
              type="text"
              placeholder={t("auth.login.form.username.placeholder")}
              value={username()}
              onInput={(e) => setUsername(e.currentTarget.value)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
              autocomplete="username"
            />
          </label>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="lock" size="sm" />
              {t("auth.login.form.password.label")}
            </span>
            <input
              type="password"
              placeholder={t("auth.login.form.password.placeholder")}
              value={password()}
              onInput={(e) => setPassword(e.currentTarget.value)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
              autocomplete="current-password"
            />
          </label>
        </fieldset>

        <button
          type="submit"
          disabled={isLoading()}
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <Show when={isLoading()} fallback={t("auth.login.submit")}>
            <Icon name="spinner" size="sm" class="animate-spin" />
            {t("auth.login.submitting")}
          </Show>
        </button>

        <button
          type="button"
          onClick={() => props.onShowRegister(username())}
          disabled={isLoading()}
          class="px-4 py-2 text-muted-foreground hover:text-foreground transition-colors text-sm"
        >
          {t("auth.login.noAccount")}
        </button>
      </form>
    </section>
  );
};
