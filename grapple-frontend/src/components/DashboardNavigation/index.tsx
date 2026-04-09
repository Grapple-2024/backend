import React from 'react';
import styles from './DashboardNavigation.module.css';

interface SidebarItem {
  title: string;
  active: boolean;
}

interface DashboardNavigationProps {
  sidebarData: SidebarItem[];
  onChange: (selectedTitle: string) => void;
}

const DashboardNavigation: React.FC<DashboardNavigationProps> = ({ sidebarData, onChange }) => {
  return (
    <div className={styles.dashboardNavigation}>
      {sidebarData.map((item, index) => (
        <div
          key={index}
          className={`${styles.navItem} ${item.active ? styles.active : ''}`}
          onClick={() => onChange(item.title)}
        >
          {item.title}
        </div>
      ))}
    </div>
  );
};

export default DashboardNavigation;
