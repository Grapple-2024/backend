import React, { useEffect, useState } from 'react';
import styles from './DashboardTabs.module.css';
import { set } from 'react-datepicker/dist/date_utils';

interface Tab {
  label: string;
  content: React.ReactNode;
}

interface DashboardTabsProps {
  tabs: Tab[];
  overrideTab?: number;
  setOverrideTab?: (value: any) => void;
}

const DashboardTabs: React.FC<DashboardTabsProps> = ({ tabs, overrideTab, setOverrideTab }) => {
  const [activeTab, setActiveTab] = useState<number>(0);

  const handleTabClick = (index: number) => {
    setActiveTab(index);
    setOverrideTab && setOverrideTab(undefined);
  };

  useEffect(() => {
    if (overrideTab !== undefined) {
      setActiveTab(overrideTab);
    }
    
  }, [overrideTab, activeTab]);

  return (
    <div style={{ margin: 0, padding: 0, }}>
      <div className={styles.tabContainer}>
        {tabs.map((tab, index) => (
          <div
            key={index}
            className={`${styles.tab} ${activeTab === index ? styles.activeTab : ''}`}
            onClick={() => handleTabClick(index)}
          >
            {tab.label} 
          </div>
        ))}
      </div>
      <div className={styles.tabContent}>
        {tabs[activeTab]?.content}
      </div>
    </div>
  );
};

export default DashboardTabs;
