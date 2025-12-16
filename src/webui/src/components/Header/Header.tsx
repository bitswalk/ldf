import type { Component } from "solid-js";
import { Show } from "solid-js";
import { Icon } from "../Icon";
import { Badge } from "../Badge";
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
    <header class="h-full w-full border-b flex items-center justify-between bg-card z-[100] relative">
      <section class="flex items-center h-full">
        {/* Logo */}
        <section class="w-12 h-full bg-muted flex items-center justify-center border-r border-border shrink-0">
          <span class="text-muted-foreground text-xs font-mono">LOGO</span>
        </section>
        <h1 class="text-2xl font-bold px-6">Linux Distribution Factory</h1>
      </section>
      <section class="px-6">
        <Show
          when={props.isLoggedIn && props.user}
          fallback={<Badge onClick={props.onBadgeClick} isLoggedIn={false} />}
        >
          <DropdownMenu>
            <DropdownMenuTrigger class="w-8 h-8 rounded-full bg-muted hover:border-primary transition-colors flex items-center justify-center overflow-hidden border-2 border-border group">
              <Icon
                name="user"
                size="lg"
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
      </section>
    </header>
  );
};
