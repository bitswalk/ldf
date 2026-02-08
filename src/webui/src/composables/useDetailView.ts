import { createSignal } from "solid-js";

export interface NotificationState {
  type: "success" | "error";
  message: string;
}

export function useDetailView() {
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);
  const [notification, setNotification] = createSignal<NotificationState | null>(null);
  const [editModalOpen, setEditModalOpen] = createSignal(false);
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [isSubmitting, setIsSubmitting] = createSignal(false);
  const [isDeleting, setIsDeleting] = createSignal(false);

  const showNotification = (type: "success" | "error", message: string) => {
    setNotification({ type, message });
    setTimeout(() => setNotification(null), type === "success" ? 3000 : 5000);
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  const openEditModal = () => setEditModalOpen(true);
  const closeEditModal = () => setEditModalOpen(false);
  const openDeleteModal = () => setDeleteModalOpen(true);
  const closeDeleteModal = () => setDeleteModalOpen(false);

  return {
    // State
    loading, setLoading,
    error, setError,
    notification,
    editModalOpen,
    deleteModalOpen,
    isSubmitting, setIsSubmitting,
    isDeleting, setIsDeleting,
    // Helpers
    showNotification,
    formatDate,
    // Modal controls
    openEditModal, closeEditModal,
    openDeleteModal, closeDeleteModal,
  };
}
