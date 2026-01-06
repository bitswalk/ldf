import type { Component, JSX } from "solid-js";
import { createEffect, onCleanup, Show } from "solid-js";
import { Icon } from "../Icon";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  children: JSX.Element;
}

export const Modal: Component<ModalProps> = (props) => {
  let panelRef: HTMLElement | undefined;

  createEffect(() => {
    if (props.isOpen) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
  });

  onCleanup(() => {
    document.body.style.overflow = "";
  });

  const handleBackdropClick = (e: MouseEvent) => {
    if (e.target === e.currentTarget) {
      props.onClose();
    }
  };

  return (
    <>
      <Show when={props.isOpen}>
        <section
          class="fixed inset-0 z-[99] flex justify-end"
          onClick={handleBackdropClick}
          style={{ "pointer-events": "auto" }}
        >
          {/* Backdrop with blur */}
          <aside class="absolute inset-0 bg-background/80 backdrop-blur-sm" />

          {/* Side panel */}
          <article
            ref={panelRef}
            class="relative w-[90vw] md:w-[50vw] lg:w-[40vw] bg-card border-l border-border shadow-lg flex flex-col mt-12"
            style={{
              animation: "slideInRight 0.3s ease-out",
              height: "calc(100% - 3rem)",
            }}
          >
            {/* Header */}
            <header class="flex-shrink-0 flex items-center justify-between p-6 border-b border-border">
              <h2 class="text-xl font-semibold">{props.title || "Modal"}</h2>
              <button
                onClick={props.onClose}
                class="p-2 rounded-md hover:bg-muted transition-colors"
                title="Close"
              >
                <Icon name="x" size="sm" />
              </button>
            </header>

            {/* Content */}
            <section class="flex-1 p-6 overflow-y-auto min-h-0">
              {props.children}
            </section>
          </article>
        </section>
      </Show>

      <style>
        {`
          @keyframes slideInRight {
            from {
              transform: translateX(100%);
            }
            to {
              transform: translateX(0);
            }
          }
        `}
      </style>
    </>
  );
};
