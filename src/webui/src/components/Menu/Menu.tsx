import type { Component, JSX } from "solid-js";

type MenuOrientation = "vertical" | "horizontal";

interface MenuProps {
  orientation?: MenuOrientation;
  children?: JSX.Element;
}

export const Menu: Component<MenuProps> = (props) => {
  const orientation = () => props.orientation ?? "vertical";

  const isVertical = () => orientation() === "vertical";

  return (
    <nav
      class="bg-card border-border flex"
      classList={{
        "flex-col w-[5vw] h-screen border-r": isVertical(),
        "flex-row h-[5vh] w-full border-b": !isVertical(),
      }}
    >
      {/* Logo Placeholder */}
      <section
        class="bg-muted flex items-center justify-center border-border"
        classList={{
          "w-[5vw] h-[5vw] border-b": isVertical(),
          "w-[5vh] h-[5vh] border-r": !isVertical(),
        }}
      >
        <span class="text-muted-foreground text-xs font-mono">LOGO</span>
      </section>

      {/* Menu Content */}
      <section
        class="flex-1"
        classList={{
          "flex flex-col": isVertical(),
          "flex flex-row": !isVertical(),
        }}
      >
        {props.children}
      </section>
    </nav>
  );
};
