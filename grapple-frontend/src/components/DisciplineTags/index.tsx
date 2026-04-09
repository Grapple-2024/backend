import styles from './DisciplineTags.module.css';
import { disciplineMapper, disciplines } from '@/util/default-values';

const DisciplineTags = ({ disciplinesValues }: any) => {  
  const getDisciplineStyles = (discipline: string) => {
    const disciplineData = disciplines.find((d: any) => d.label === discipline);
    return {
      backgroundColor: disciplineData?.backgroundColor || '#CCC', // Default color if not found
      color: 'black',
      padding: '5px 5px',
      fontWeight: 'bold',
      borderRadius: '5px',
      fontSize: 10,
      marginRight: '5px',
      display: 'flex',
      alignItems: 'center',
    };
  };

  return (
    <div className={styles.disciplines}>
      {disciplinesValues?.map((discipline: string) => (
        <span key={discipline} style={getDisciplineStyles(discipline)}>
          {disciplineMapper(discipline)}
        </span>
      ))}
    </div>
  );
};

export default DisciplineTags;