import type { Component } from "solid-js";
import { createSignal, Show } from "solid-js";
import { Icon } from "../../components/Icon";
import type { APIInfo } from "../../services/storage";

interface ConnectionProps {
  onConnect: (serverUrl: string, apiInfo: APIInfo) => void;
  initialError?: string | null;
}

export const Connection: Component<ConnectionProps> = (props) => {
  const [serverUrl, setServerUrl] = createSignal("");
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(
    props.initialError ?? null,
  );

  const normalizeUrl = (url: string): string => {
    return url.replace(/\/+$/, "");
  };

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setError(null);
    setIsLoading(true);

    const baseUrl = normalizeUrl(serverUrl());

    try {
      const response = await fetch(`${baseUrl}/`, {
        method: "GET",
        headers: {
          Accept: "application/json",
        },
      });

      if (response.ok) {
        const apiInfo: APIInfo = await response.json();

        if (apiInfo.name === "ldfd" && apiInfo.endpoints?.auth) {
          props.onConnect(baseUrl, apiInfo);
        } else {
          setError("Server responded but does not appear to be an LDF server.");
        }
      } else {
        setError("Server responded with an error. Please verify the URL.");
      }
    } catch {
      setError(
        "Unable to connect to server. Please check the URL and try again.",
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <section class="h-full flex flex-col items-center justify-center p-8">
      <h1 class="text-4xl font-bold mb-2">Connect to Server</h1>
      <p class="text-muted-foreground mb-8">
        Enter the LDF server URL to get started
      </p>

      <form onSubmit={handleSubmit} class="w-full max-w-md flex flex-col gap-4">
        <Show when={error()}>
          <aside class="p-3 bg-destructive/10 border border-destructive/20 rounded-md text-destructive text-sm">
            {error()}
          </aside>
        </Show>

        <fieldset class="flex flex-col gap-4" disabled={isLoading()}>
          <legend class="sr-only">Server Connection</legend>

          <label class="flex flex-col gap-1">
            <span class="text-sm text-muted-foreground flex items-center gap-2">
              <Icon name="plugs" size="sm" />
              Server URL
            </span>
            <input
              type="url"
              placeholder="http://localhost:8443"
              value={serverUrl()}
              onInput={(e) => setServerUrl(e.currentTarget.value)}
              class="px-4 py-2 border border-border rounded-md bg-background focus:outline-none focus:ring-2 focus:ring-primary disabled:opacity-50"
              required
            />
            <span class="text-xs text-muted-foreground">
              The URL of your LDF server instance
            </span>
          </label>
        </fieldset>

        <button
          type="submit"
          disabled={isLoading()}
          class="px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
        >
          <Show when={isLoading()} fallback="Connect">
            <Icon name="spinner" size="sm" class="animate-spin" />
            Discovering API...
          </Show>
        </button>
      </form>
    </section>
  );
};
