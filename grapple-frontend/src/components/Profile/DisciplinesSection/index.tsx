import { useState } from 'react';
import Select from 'react-select';
import styles from './DisciplinesSection.module.css'

import { CiEdit } from "react-icons/ci";

const selections = [
  { value: "jiu-jitsu", label: "Jiu-Jitsu" },
  { value: "boxing", label: "Boxing" },
  { value: "wrestling", label: "Wrestling" },
  { value: "mma", label: "MMA" },
  { value: "karate", label: "Karate" },
  { value: "muay-thai", label: "Muay Thai" },
  { value: "judo", label: "Judo" },
  { value: "taekwondo", label: "Taekwondo" }
];

interface DisciplinesSectionProps {
  isCoach?: boolean;
  gym: any;
  updateGym?: any;
}

const DisciplinesSection = ({
  isCoach = false,
  gym,
  updateGym,
}: DisciplinesSectionProps) => {
  const [isEditingDisciplines, setIsEditingDisciplines] = useState(false);
  const [newDisciplines, setNewDisciplines] = useState<any[]>([]);
  
  if (isCoach) {
    return (
      <>
        <div className={styles.sectionHeader}>
          <h2>Gym Disciplines</h2>
          <CiEdit 
            size={20} 
            className={styles.editIcon}
            style={{ marginBottom: '1rem' }}
            onClick={() => {
              setIsEditingDisciplines(true);
              setNewDisciplines(gym?.disciplines?.map((d: any) => ({ value: d, label: d })) || []);
            }}
          />
        </div>
        {isEditingDisciplines ? (
          <div className={styles.editDisciplinesContainer}>
            <Select
              isMulti
              name="Disciplines"
              options={selections}
              value={newDisciplines}
              classNamePrefix="select"
              className={styles.disciplinesSelect}
              onChange={(e) => setNewDisciplines(e as any)}
            />
            <div className={styles.editActions}>
              <button 
                className={styles.cancelButton}
                onClick={() => setIsEditingDisciplines(false)}
              >
                Cancel
              </button>
              <button 
                className={styles.saveButton}
                onClick={() => {
                  updateGym.mutate({
                    ...gym,
                    disciplines: newDisciplines.map((d: any) => d.value),
                  });
                  setIsEditingDisciplines(false);
                }}
              >
                Save Changes
              </button>
            </div>
          </div>
        ) : (
          <ul className={styles.disciplinesList}>
            {gym?.disciplines?.map((discipline: string) => (
              <li key={discipline} className={styles.disciplineItem}>
                {discipline}
              </li>
            ))}
          </ul>
        )}
      </>
    )
  }

  return (
    <>
      <div className={styles.sectionHeader}>
        <h2>Gym Disciplines</h2>
      </div>
      <ul className={styles.disciplinesList}>
        {gym?.disciplines?.map((discipline: string) => (
          <li key={discipline} className={styles.disciplineItem}>
            {discipline}
          </li>
        ))}
      </ul>
    </>
  )
};

export default DisciplinesSection;