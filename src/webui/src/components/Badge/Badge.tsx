import type { Component } from "solid-js";
import { Icon } from "../Icon";
import { t } from "../../services/i18n";

interface BadgeProps {
  onClick?: () => void;
  avatarUrl?: string;
  isLoggedIn?: boolean;
}

export const Badge: Component<BadgeProps> = (props) => {
  return (
    <button
      onClick={props.onClick}
      class="w-8 h-8 rounded-full bg-muted hover:border-primary transition-colors flex items-center justify-center overflow-hidden border-2 border-border group"
      title={
        props.isLoggedIn ? t("auth.header.profile") : t("auth.login.title")
      }
    >
      {props.avatarUrl ? (
        <img
          src={props.avatarUrl}
          alt="Avatar"
          class="w-full h-full object-cover"
        />
      ) : (
        <Icon
          name="user"
          size="lg"
          class="text-muted-foreground group-hover:text-primary transition-colors"
        />
      )}
    </button>
  );
};
