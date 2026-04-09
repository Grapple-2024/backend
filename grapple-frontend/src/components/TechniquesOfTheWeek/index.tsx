import React, { useState } from 'react';
import styles from './TechniquesOfTheWeek.module.css';
import TechniqueCard from '../TechniqueCard';
import EditableCard from '../EditableCard';
import { Row } from 'react-bootstrap';
import SeriesSearchCard from '../SeriesSearchCard';

interface TechniquesOfTheWeekProps {
  series: any[];
  onSave?: any;
  create: boolean;
  path?: string;
  quickAction?: boolean;
  onDelete?: (id: string) => void;
  daySelected?: Date;
}

const TechniquesOfTheWeek: React.FC<TechniquesOfTheWeekProps> = ({ 
  series, 
  onSave,
  quickAction = false,
  create = false,
  path = '/coach/content',
  onDelete,
  daySelected,
}) => {
  const [newSeriesAdded, setNewSeriesAdded] = useState<string>("Search for a Series");
  
  return (
    <div className={styles.seriesWrapperContainer}>
      <div className={styles.seriesList}>
        {create && (
          <div className={styles.createRow} style={{ marginLeft: 20, marginRight: 20 }}>
            <EditableCard
              createText="Add New Series"
              quickAction={quickAction}
              editComponent={
                <SeriesSearchCard 
                  onSearch={(searchTerm) => setNewSeriesAdded(searchTerm)}
                  onSave={onSave && onSave}
                  searchValue={newSeriesAdded}
                  onCancel={() => {}}
                  daySelected={daySelected}
                />
              }
            />
          </div>
        )}
        {series?.length === 0 ? (
          <div className={styles.noSeries}>
            <p>Nothing posted this week. Please check back later.</p>
          </div>
        ) : (
          series?.map((seriesItem) => (
            <Row style={{ margin: "0px 20px 20px", padding: 0 }} key={seriesItem.id}>
              <TechniqueCard
                techniqueId={seriesItem.id}
                seriesItem={seriesItem.series}
                onDelete={onDelete}
                path={path}
              />
            </Row>
          ))
        )}
      </div>
    </div>
  );
};

export default TechniquesOfTheWeek;
