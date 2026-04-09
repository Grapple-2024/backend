import React from 'react';
import { FaClock } from 'react-icons/fa';
import { TfiAnnouncement } from "react-icons/tfi";
import styles from './QuickActions.module.css';

const QuickActions = ({ actions }: any) => {
  
  return (
    <div className={styles.quickActionsContainer}>
      <h4 className={styles.title}>Quick Actions</h4>
      <div className={styles.actionsList}>
        {actions.map((action: any, index: number) => (
          <div key={index} className={styles.actionItem} onClick={action?.override}>
            <div className={styles.icon}>{action?.icon}</div>
            <p className={styles.label}>{action?.label}</p>
          </div>
        ))}
      </div>
    </div>
  );
};

export default QuickActions;
