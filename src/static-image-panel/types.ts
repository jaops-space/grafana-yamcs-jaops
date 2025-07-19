export interface ImagePanelOptions {
    manualImageUrl: boolean;
    imageUrl: string;
    objectFit?: 'fill' | 'contain' | 'cover' | 'none';
    grayscale?: string;
    sepia?: string;
    brightness?: number;
    contrast?: number;
    transform?: string; // e.g., "translate(10px, 20px) rotate(45deg) scale(1.2)"
}
