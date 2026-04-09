import { CiEdit } from 'react-icons/ci';
import styles from './DescriptionSection.module.css';
import { useState } from 'react';

interface DescriptionSectionProps {
  isCoach?: boolean;
  gym: any;
  updateGym?: any;
}

const DescriptionSection = ({
  isCoach = false,
  gym,
  updateGym,
}: DescriptionSectionProps) => {
  const [isUpdatingDescription, setIsUpdatingDescription] = useState(false);
  const [newDescription, setNewDescription] = useState<string>('');
  const [showFullDescription, setShowFullDescription] = useState(false);

  if (isCoach) {
    return (
      <>
        <div className={styles.sectionHeader}>
          <h2>About Us</h2>
          <CiEdit size={20} className={styles.editIcon} style={{ marginBottom: '1.1rem' }} onClick={() => {
            setIsUpdatingDescription(true);
            setNewDescription(gym?.description);
          }} />
        </div>
        {isUpdatingDescription ? (
          <div className={styles.editDescriptionContainer}>
            <textarea
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
              className={styles.editTextarea}
              rows={5}
            />
            <div className={styles.editActions}>
              <button 
                className={styles.cancelButton}
                onClick={() => setIsUpdatingDescription(false)}
              >
                Cancel
              </button>
              <button 
                className={styles.saveButton}
                onClick={() => {
                  updateGym.mutate({
                    ...gym,
                    description: newDescription,
                  })
                  setIsUpdatingDescription(false);
                }}
              >
                Save Changes
              </button>
            </div>
          </div>
        ) : (
          <>
            <p className={showFullDescription ? styles.fullDescription : styles.description}>
              {gym?.description}
            </p>
            {gym?.description?.length > 200 && (
              <button 
                className={styles.readMoreButton}
                onClick={() => setShowFullDescription(!showFullDescription)}
              >
                {showFullDescription ? 'Read less' : 'Read more'}
              </button>
            )}
          </>
        )}
      </>
    );
  }

  return (
    <>
      <div className={styles.sectionHeader}>
        <h2>About Us</h2>
      </div>
      <p className={showFullDescription ? styles.fullDescription : styles.description}>
        {gym?.description}
      </p>
      {gym?.description?.length > 200 && (
        <button 
          className={styles.readMoreButton}
          onClick={() => setShowFullDescription(!showFullDescription)}
        >
          {showFullDescription ? 'Read less' : 'Read more'}
        </button>
      )}
    </>
  );
};

export default DescriptionSection;