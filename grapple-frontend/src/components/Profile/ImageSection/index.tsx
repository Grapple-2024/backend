import Image from 'next/image';
import styles from './ImageSection.module.css';
import { CiEdit } from 'react-icons/ci';

interface ImageSectionProps {
  isCoach?: boolean;
  bannerUrl: string;
  setEditingImage?: (imageType: string) => void;
  setIsModalOpen?: (isOpen: boolean) => void;
}

const ImageSection = ({
  isCoach = false,
  bannerUrl,
  setEditingImage,
  setIsModalOpen,
}: ImageSectionProps) => {

  if (isCoach) {
    return (
      <div className={styles.imageSection}>
        <div className={styles.imageContainer}>
          <Image 
            src={bannerUrl || '/placeholder-banner.png'}
            alt="Gym Banner"
            fill
            className={styles.bannerImage}
            priority
          />
          <div className={styles.editButton} onClick={() => {
            setEditingImage && setEditingImage('banner');
            setIsModalOpen && setIsModalOpen(true);
          }}>
            <CiEdit color="#1E1E1E" size={20}/>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.imageSection}>
      <div className={styles.imageContainer}>
        <Image 
          src={bannerUrl || '/placeholder-banner.png'}
          alt="Gym Banner"
          fill
          className={styles.bannerImage}
          priority
        />
      </div>
    </div>
  )
};

export default ImageSection;