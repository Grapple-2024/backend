import React, { useState, useEffect } from 'react';
import { FaChevronLeft, FaChevronRight } from 'react-icons/fa';
import styles from './DatePicker.module.css';

interface DatePickerProps {
  selectedDate?: Date;
  onChange?: (startDayOfWeek: Date) => void;
  onDaySelect?: (date: Date) => void;
  isProfilePage?: boolean;
}

const daysOfWeek = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

const DatePicker: React.FC<DatePickerProps> = ({ 
  selectedDate = new Date(), 
  onChange,
  onDaySelect,
  isProfilePage = false,
}) => {
  const [currentDate, setCurrentDate] = useState<Date>(selectedDate);
  const [disableNext, setDisableNext] = useState<boolean>(false);

  useEffect(() => {
    if (onChange) {
      onChange(currentDate);
    }

    if (!isProfilePage) {
      const today = new Date();
      today.setHours(0, 0, 0, 0);

      const dayOfWeek = today.getDay();

      const nextMonday = new Date(today);
      if (dayOfWeek !== 0) {
        nextMonday.setDate(today.getDate() + (7 - dayOfWeek + 1));
      } else {
        nextMonday.setDate(today.getDate() + 1);
      }
      nextMonday.setHours(0, 0, 0, 0);

      const nextSunday = new Date(nextMonday);
      nextSunday.setDate(nextMonday.getDate() + 6);
      nextSunday.setHours(0, 0, 0, 0);

      const todayPlus7 = new Date(currentDate);
      todayPlus7.setDate(todayPlus7.getDate() + 7);
      todayPlus7.setHours(0, 0, 0, 0);

      const isInNextWeek = todayPlus7 >= nextMonday && todayPlus7 <= nextSunday;

      if (currentDate < today && !isInNextWeek) {
        setDisableNext(false);
      } else {
        setDisableNext(true);
      }
    } else {
      setDisableNext(false); // Never disable next for profile page
    }
  }, [currentDate, onChange, isProfilePage]);
  
  const startOfWeek = (date: Date): Date => {
    const day = date.getDay();
    const diff = date.getDate() - day + (day === 0 ? -6 : 1);
    return new Date(date.setDate(diff));
  };

  const handlePrevWeek = () => {
    const prevWeek = new Date(currentDate);
    prevWeek.setDate(prevWeek.getDate() - 7);
    setCurrentDate(startOfWeek(prevWeek));
    onDaySelect && onDaySelect(startOfWeek(prevWeek));
  };

  const handleNextWeek = () => {
    if (isProfilePage || !disableNext) {
      const nextWeek = new Date(currentDate);
      nextWeek.setDate(nextWeek.getDate() + 7);
      setCurrentDate(startOfWeek(nextWeek));
      onDaySelect && onDaySelect(startOfWeek(nextWeek));
    }
  };

  const handleDayClick = (date: Date) => {
    const today = new Date();
    if (isProfilePage || date <= today) {
      setCurrentDate(date);
      onDaySelect && onDaySelect(date);
    }
  };

  const renderDaysOfWeek = () => {
    const start = startOfWeek(new Date(currentDate));
    const today = new Date();
    
    return daysOfWeek.map((day, index) => {
      const date = new Date(start);
      date.setDate(start.getDate() + index);
      const isSelected = date.toDateString() === currentDate.toDateString();
      const isFuture = date > today;
      
      return (
        <div
          key={day}
          className={`${styles.day} ${isSelected ? styles.selected : ''} ${
            !isProfilePage && isFuture ? styles.disabled : ''
          }`}
          onClick={() => (isProfilePage || !isFuture) && handleDayClick(date)}
        >
          <span className={styles.dayName}>{day}</span>
          <span className={styles.dayDate}>{date.getDate()}</span>
        </div>
      );
    });
  };

  return (
    <div className={styles.weekPicker}>
      <div className={styles.header}>
        <div className={styles.monthYear}>
          {currentDate.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
        </div>
        <div className={styles.navIcons}>
          <FaChevronLeft onClick={handlePrevWeek} className={styles.navIcon} />
          <FaChevronRight 
            onClick={handleNextWeek} 
            className={`${styles.navIcon} ${!isProfilePage && disableNext ? styles.disabledIcon : ''}`} 
          />
        </div>
      </div>
      <div className={styles.days}>{renderDaysOfWeek()}</div>
    </div>
  );
};

export default DatePicker;