import { createSignal, createEffect, createRoot } from "solid-js";
import { debugError } from "../lib/utils";

export type TimeOfDay = "day" | "night";

class TimeService {
  private timeOfDay: ReturnType<typeof createSignal<TimeOfDay>>;
  private checkInterval: number | null = null;

  constructor() {
    // Wrap signal creation in createRoot to avoid disposal warnings
    this.timeOfDay = createRoot(() => {
      return createSignal<TimeOfDay>("day");
    });
  }

  getTimeOfDay = () => this.timeOfDay[0]();

  private determineTimeOfDay(): TimeOfDay {
    try {
      const hour = new Date().getHours();
      // Night from 18:00 to 06:00, day otherwise
      const isNight = hour >= 18 || hour < 6;
      return isNight ? "night" : "day";
    } catch (error) {
      debugError("[TimeService] Failed to get current time:", error);
      // Fallback to day mode if we can't determine time
      return "day";
    }
  }

  private updateTimeOfDay(): void {
    const newTimeOfDay = this.determineTimeOfDay();
    const currentTimeOfDay = this.timeOfDay[0]();

    if (newTimeOfDay !== currentTimeOfDay) {
      this.timeOfDay[1](newTimeOfDay);
    }
  }

  initialize(): void {
    // Set initial time of day
    this.updateTimeOfDay();

    // Check every minute for time changes
    this.checkInterval = window.setInterval(() => {
      this.updateTimeOfDay();
    }, 60000); // Check every minute
  }

  cleanup(): void {
    if (this.checkInterval !== null) {
      window.clearInterval(this.checkInterval);
      this.checkInterval = null;
    }
  }

  // Subscribe to time of day changes
  onTimeOfDayChange(callback: (timeOfDay: TimeOfDay) => void): () => void {
    return createEffect(() => {
      callback(this.timeOfDay[0]());
    });
  }
}

export const timeService = new TimeService();
