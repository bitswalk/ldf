import type { Component } from "solid-js";
import { Show } from "solid-js";
import { Icon } from "../Icon";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../DropdownMenu";

interface UserInfo {
  name: string;
  email: string;
  role: string;
}

interface HeaderProps {
  isLoggedIn?: boolean;
  user?: UserInfo | null;
  onLogout?: () => void;
  onSettings?: () => void;
  onBadgeClick?: () => void;
}

export const Header: Component<HeaderProps> = (props) => {
  return (
    <header class="h-full w-full border-b flex items-center justify-between px-6">
      <h1 class="text-2xl font-bold">Linux Distribution Factory</h1>
      <Show
        when={props.isLoggedIn && props.user}
        fallback={
          <button
            onClick={props.onBadgeClick}
            class="w-10 h-10 rounded-full bg-muted hover:border-primary transition-colors flex items-center justify-center overflow-hidden border-2 border-border group"
            title="Login"
          >
            <Icon
              name="user"
              size="xl"
              class="text-muted-foreground group-hover:text-primary transition-colors"
            />
          </button>
        }
      >
        <DropdownMenu>
          <DropdownMenuTrigger class="w-10 h-10 rounded-full bg-muted hover:border-primary transition-colors flex items-center justify-center overflow-hidden border-2 border-border group">
            <Icon
              name="user"
              size="xl"
              class="text-muted-foreground group-hover:text-primary transition-colors"
            />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" side="bottom">
            <DropdownMenuLabel>
              <section class="flex flex-col">
                <span class="font-medium">{props.user?.name}</span>
                <span class="text-xs text-muted-foreground">
                  {props.user?.email}
                </span>
              </section>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={props.onSettings} class="gap-2">
              <Icon name="gear" size="sm" />
              <span>Settings</span>
            </DropdownMenuItem>
            <DropdownMenuItem
              onSelect={props.onLogout}
              class="gap-2 text-destructive focus:text-destructive"
            >
              <Icon name="sign-out" size="sm" />
              <span>Logout</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </Show>
    </header>
  );
};
