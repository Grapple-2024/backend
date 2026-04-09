import styles from '../MemberDrawer.module.css';

export default function NotesTab() {
  return (
    <div className={styles.tabContent}>
      <div className={styles.comingSoon}>
        <span className={styles.comingSoonIcon}>📝</span>
        <p className={styles.comingSoonText}>Member notes coming soon.</p>
      </div>
    </div>
  );
}
