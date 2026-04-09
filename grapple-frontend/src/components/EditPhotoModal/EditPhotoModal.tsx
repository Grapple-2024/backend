import { Modal } from 'react-bootstrap';
import { useRef, useState, useCallback } from 'react';
import { FaUpload } from 'react-icons/fa';
import Cropper from 'react-easy-crop';
import styles from './EditPhotoModal.module.css';

interface CropDimensions {
  width: number;
  height: number;
  aspect: number;
}

// Predefined crop dimensions for different image types
const CROP_PRESETS: Record<'banner' | 'logo' | 'hero' | 'avatar', CropDimensions> = {
  banner: {
    width: 1920,
    height: 400,
    aspect: 1920 / 400,
  },
  logo: {
    width: 400,
    height: 400,
    aspect: 1,
  },
  hero: {
    width: 1920,
    height: 400,
    aspect: 1920 / 400,
  },
  avatar: {
    width: 400,
    height: 400,
    aspect: 1,
  },
};

interface EditPhotoModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (newPhoto: Blob, name: string) => void;
  imageType: 'banner' | 'logo' | 'hero' | 'avatar';
}

interface CropArea {
  x: number;
  y: number;
  width: number;
  height: number;
}

const EditPhotoModal = ({ isOpen, onClose, onSave, imageType }: EditPhotoModalProps) => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<string | null>(null);
  const [crop, setCrop] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [croppedAreaPixels, setCroppedAreaPixels] = useState<CropArea | null>(null);

  // Get crop dimensions based on image type
  const cropDimensions = CROP_PRESETS[imageType];

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const file = e.target.files[0];
      setSelectedFile(file);
      const reader = new FileReader();
      reader.onloadend = () => {
        setPreview(reader.result as string);
      };
      reader.readAsDataURL(file);
    }
  };

  const onCropComplete = useCallback((croppedArea: any, croppedAreaPixels: CropArea) => {
    setCroppedAreaPixels(croppedAreaPixels);
  }, []);

  const getCroppedImg = async (
    imageSrc: string,
    pixelCrop: CropArea
  ): Promise<Blob> => {
    return new Promise((resolve, reject) => {
      const image = document.createElement('img');
      image.src = imageSrc;
      
      image.onload = () => {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');

        if (!ctx) {
          reject(new Error('No 2d context'));
          return;
        }

        canvas.width = cropDimensions.width;
        canvas.height = cropDimensions.height;

        ctx.drawImage(
          image,
          pixelCrop.x,
          pixelCrop.y,
          pixelCrop.width,
          pixelCrop.height,
          0,
          0,
          cropDimensions.width,
          cropDimensions.height
        );

        canvas.toBlob(
          (blob) => {
            if (blob) {
              resolve(blob);
            } else {
              reject(new Error('Canvas to Blob conversion failed'));
            }
          },
          'image/png',
          0.95
        );
      };

      image.onerror = () => {
        reject(new Error('Image loading failed'));
      };
    });
  };

  const handleSave = async () => {
    if (preview && croppedAreaPixels) {
      try {
        const croppedImage = await getCroppedImg(preview, croppedAreaPixels);
        onSave(croppedImage, selectedFile?.name as string);
        handleClose();
      } catch (e) {
        console.error('Error cropping image:', e);
      }
    }
  };

  const handleClose = () => {
    setSelectedFile(null);
    setPreview(null);
    setCrop({ x: 0, y: 0 });
    setZoom(1);
    setCroppedAreaPixels(null);
    onClose();
  };

  return (
    <Modal 
      show={isOpen} 
      onHide={handleClose} 
      centered 
      dialogClassName={styles.modalContent}
    >
      <Modal.Header closeButton className={styles.modalHeader}>
        <Modal.Title className={styles.modalTitle}>
          Change {imageType?.charAt(0)?.toUpperCase() + imageType?.slice(1)} Image
        </Modal.Title>
      </Modal.Header>
      <Modal.Body className={styles.uploadContainer}>
        <input
          type="file"
          ref={fileInputRef}
          onChange={handleFileChange}
          accept="image/*"
          className={styles.hiddenInput}
        />
        
        {preview ? (
          <div 
            className={styles.previewContainer}
            style={{ height: '400px' }}
          >
            <Cropper
              image={preview}
              crop={crop}
              zoom={zoom}
              aspect={cropDimensions?.aspect}
              onCropChange={setCrop}
              onZoomChange={setZoom}
              onCropComplete={onCropComplete}
              objectFit="contain"
              restrictPosition={false}
            />
          </div>
        ) : (
          <div
            className={styles.dropZone}
            onClick={() => fileInputRef.current?.click()}
          >
            <FaUpload size={24} color="#666" />
            <p className={styles.dropZoneText}>Click to upload a photo</p>
            <p className={styles.dropZoneSubtext}>
              Recommended size: {cropDimensions?.width}x{cropDimensions?.height}px
            </p>
          </div>
        )}

        {preview && (
          <div style={{ marginTop: '1rem' }}>
            <input
              type="range"
              value={zoom}
              min={1}
              max={3}
              step={0.1}
              aria-labelledby="zoom"
              onChange={(e) => setZoom(parseFloat(e.target.value))}
              style={{ width: '100%' }}
            />
          </div>
        )}
      </Modal.Body>
      <Modal.Footer className={styles.modalFooter}>
        <button
          className={styles.secondaryButton}
          onClick={handleClose}
        >
          Cancel
        </button>
        <button
          className={styles.primaryButton}
          onClick={handleSave}
          disabled={!preview}
        >
          Save Changes
        </button>
      </Modal.Footer>
    </Modal>
  );
};

export default EditPhotoModal;