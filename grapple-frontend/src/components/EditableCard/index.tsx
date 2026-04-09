import React, { useEffect, useState } from 'react';
import { FaPlus } from 'react-icons/fa';
import styles from './EditableCard.module.css';

interface EditableCardProps {
  createText: string;
  editComponent: React.ReactNode;
  quickAction: boolean;
}

const EditableCard: React.FC<EditableCardProps> = ({ createText, editComponent, quickAction = false }) => {
const [isEditing, setIsEditing] = useState<any>(null);

  useEffect(() => {
    if (!isEditing) {
      setIsEditing(quickAction);
    }
  }, [quickAction]);
  
  return (
    <div>
      {!isEditing ? (
        <div
          className={styles.createCard}
          onClick={() => setIsEditing(true)}
        >
          <FaPlus size={24} className={styles.plusIcon} />
          <p className={styles.createText}>{createText}</p>
        </div>
      ) : (
        <div className={styles.editCard}>
          {React.cloneElement(editComponent as React.ReactElement<any>, {
            onCancel: () => setIsEditing(false),
          })}
        </div>
      )}
    </div>
  );
};

export default EditableCard;
