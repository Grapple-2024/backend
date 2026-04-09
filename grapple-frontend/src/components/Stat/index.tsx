import React from 'react';
import styles from './Stat.module.css';

interface StatProps {
  title: string;
  value: string | number;
  subtitle?: string;
}

const Stat: React.FC<StatProps> = ({ title, value, subtitle }) => {
  return (
    <div className={styles.statContainer}>
      <div className={styles.title}>{title}</div>
      <div className={styles.value}>{value}</div>
      <div className={styles.subtitle}>{subtitle}</div>
    </div>
  );
};

export default Stat;
