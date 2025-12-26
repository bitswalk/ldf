import type { Component } from "solid-js";
import { createSignal, For, Show } from "solid-js";
import { Icon } from "../../components/Icon";
import { createUser } from "../../services/auth";
import { t } from "../../services/i18n";

type UserRole = "root" | "developer" | "anonymous";

interface RoleOption {
  value: UserRole;
  label: string;
  description: string;
}

const ROLE_OPTIONS: RoleOption[] = [
  {
    value: "developer",
    label: "Developer",
    description: "Standard user with development access",
  },
  {
    value: "root",
    label: "Root",
    description: "Full administrative privileges",
  },
  {
    value: "anonymous",
    label: "Anonymous",
    description: "Limited read-only access",
  },
];

interface RegisterProps {
  serverUrl: string;
  prefillUsername?: string;
  onSuccess: (
    user: { id: string; name: string; email: string; role: string },
    token: string,
  ) => void;
  onBackToLogin: () => void;
}

export const Register: Component<RegisterProps> = (props) => {
  const [username, setUsername] = createSignal(props.prefillUsername || "");
  const [email, setEmail] = createSignal("");
  const [password, setPassword] = createSignal("");
  const [confirmPassword, setConfirmPassword] = createSignal("");
  const [role, setRole] = createSignal<UserRole>("developer");
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError(null);

    if (password() !== confirmPassword()) {
      setError(t("auth.register.errors.passwordMismatch"));
      return;
    }

    if (password().length < 8) {
      setError(t("auth.register.errors.weakPassword"));
      return;
    }

    setIsLoading(true);

    const result = await createUser(username(), password(), email(), role());

    setIsLoading(false);

    if (result.success) {
      props.onSuccess(result.user, result.token);
    } else {
      switch (result.error) {
        case "email_exists":
          setError(t("auth.register.errors.emailExists"));
          break;
        case "user_exists":
          setError(t("auth.register.errors.userExists"));
          break;
        case "root_exists":
          setError(t("auth.register.errors.userExists"));
          break;
        case "network_error":
          setError(t("auth.register.errors.networkError"));
          break;
        default:
          setError(result.message || t("auth.register.errors.generic"));
      }
    }
  };

  return (
    <section class="h-full flex flex-col items-center justify-center p-8">
      <h1 class="text-4xl font-bold mb-2">{t("auth.register.title")}</h1>
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
          <legend class="sr-only">{t("auth.register.title")}</legend>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="user" size="sm" />
              {t("auth.register.form.username.label")}
            </span>
            <input
              type="text"
              placeholder={t("auth.register.form.username.placeholder")}
              value={username()}
              onInput={(e) => setUsername(e.currentTarget.value)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
              autocomplete="username"
            />
          </label>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="envelope" size="sm" />
              {t("auth.register.form.email.label")}
            </span>
            <input
              type="email"
              placeholder={t("auth.register.form.email.placeholder")}
              value={email()}
              onInput={(e) => setEmail(e.currentTarget.value)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
              autocomplete="email"
            />
          </label>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="user-circle" size="sm" />
              Role
            </span>
            <select
              value={role()}
              onChange={(e) => setRole(e.currentTarget.value as UserRole)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
            >
              <For each={ROLE_OPTIONS}>
                {(option) => (
                  <option value={option.value}>{option.label}</option>
                )}
              </For>
            </select>
            <span class="text-xs text-muted-foreground">
              {ROLE_OPTIONS.find((opt) => opt.value === role())?.description}
            </span>
          </label>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="lock" size="sm" />
              {t("auth.register.form.password.label")}
            </span>
            <input
              type="password"
              placeholder={t("auth.register.form.password.placeholder")}
              value={password()}
              onInput={(e) => setPassword(e.currentTarget.value)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
              minLength={8}
              autocomplete="new-password"
            />
          </label>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="lock" size="sm" />
              {t("auth.register.form.confirmPassword.label")}
            </span>
            <input
              type="password"
              placeholder={t("auth.register.form.confirmPassword.placeholder")}
              value={confirmPassword()}
              onInput={(e) => setConfirmPassword(e.currentTarget.value)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
              minLength={8}
              autocomplete="new-password"
            />
          </label>
        </fieldset>

        <button
          type="submit"
          disabled={isLoading()}
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <Show when={isLoading()} fallback={t("auth.register.submit")}>
            <Icon name="spinner" size="sm" class="animate-spin" />
            {t("auth.register.submitting")}
          </Show>
        </button>

        <button
          type="button"
          onClick={props.onBackToLogin}
          disabled={isLoading()}
          class="px-4 py-2 text-muted-foreground hover:text-foreground transition-colors text-sm"
        >
          {t("auth.register.hasAccount")}
        </button>
      </form>
    </section>
  );
};
