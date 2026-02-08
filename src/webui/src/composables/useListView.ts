import { createSignal } from "solid-js";

export function useListView<T>() {
  const [isLoading, setIsLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);
  const [isModalOpen, setIsModalOpen] = createSignal(false);
  const [deleteModalOpen, setDeleteModalOpen] = createSignal(false);
  const [isDeleting, setIsDeleting] = createSignal(false);
  const [isSubmitting, setIsSubmitting] = createSignal(false);
  const [selected, setSelected] = createSignal<T[]>([]);
  const [itemsToDelete, setItemsToDelete] = createSignal<T[]>([]);

  const openModal = () => setIsModalOpen(true);
  const closeModal = () => setIsModalOpen(false);
  const openDeleteModal = () => setDeleteModalOpen(true);
  const closeDeleteModal = () => {
    setDeleteModalOpen(false);
    setItemsToDelete([]);
  };

  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleDateString();
  };

  return {
    isLoading, setIsLoading,
    error, setError,
    isModalOpen, openModal, closeModal,
    deleteModalOpen, openDeleteModal, closeDeleteModal,
    isDeleting, setIsDeleting,
    isSubmitting, setIsSubmitting,
    selected, setSelected,
    itemsToDelete, setItemsToDelete,
    formatDate,
  };
}
