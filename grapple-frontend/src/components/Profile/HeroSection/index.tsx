import { CiEdit } from 'react-icons/ci';
import styles from './HeroSection.module.css';

import Image from "next/image";

interface HeroSectionProps {
  isCoach?: boolean;
  gym: any;
  setEditingImage?: any;
  setIsModalOpen?: any;
};

const HeroSection = ({
  isCoach = false,
  gym,
  setEditingImage,
  setIsModalOpen,
}: HeroSectionProps) => {

  if (isCoach) {
    return (
      <>
        <div className={styles.videoContainer}>
          <Image 
            src={gym?.hero_url || '/placeholder-banner.png'}
            alt="Hero"
            fill
            className={styles.bannerImage}
            priority
          />
          <div className={styles.editButton} onClick={() => {
            setEditingImage('hero');
            setIsModalOpen(true);
          }}>
            <CiEdit color="#1E1E1E" size={20}/>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <div className={styles.videoContainer}>
        <Image 
          src={gym?.hero_url || '/placeholder-banner.png'}
          alt="Hero"
          fill
          className={styles.bannerImage}
          priority
        />
      </div>
    </>
  );
}

export default HeroSection;