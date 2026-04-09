import React, { useEffect, useState } from 'react';
import styles from './SeriesSearchCard.module.css';
import { FaEye, FaHeart, FaCalendar } from 'react-icons/fa6';
import Search from '../Search';
import gymApi from '@/util/gym-api';
import { Image, OverlayTrigger, Popover, Row } from 'react-bootstrap';
import ExpandableText from '../ExpandableText';
import DisciplineTags from '../DisciplineTags';
import { useMobileContext } from '@/context/mobile';
import DateRangePicker from '@wojtekmaj/react-daterange-picker';
import '@wojtekmaj/react-daterange-picker/dist/DateRangePicker.css';
import 'react-calendar/dist/Calendar.css';import '@wojtekmaj/react-daterange-picker/dist/DateRangePicker.css';
import 'react-calendar/dist/Calendar.css';
import './style.css';
import { useMessagingContext } from '@/context/message';
import { useToken } from '@/hook/user';
import { useGetGym } from '@/hook/gym';

interface SearchCardProps {
  searchValue: string;
  onSearch: (searchTerm: string) => void;
  onSave?: any;
  onCancel?: () => void;
  daySelected?: Date;
}
type ValuePiece = Date | null;

type Value = ValuePiece | [ValuePiece, ValuePiece];

function getCurrentWeek(daySelected?: Date): any {
  const today = daySelected ? new Date(daySelected) : new Date();
  const dayOfWeek = today.getDay(); // Sunday - Saturday: 0 - 6

  // Calculate how many days to go back to get to Monday (if today is Sunday, we go back 6 days)
  const daysSinceMonday = dayOfWeek === 0 ? 6 : dayOfWeek - 1;

  // Calculate Monday by subtracting the appropriate number of days
  const monday = new Date(today);
  monday.setDate(today.getDate() - daysSinceMonday);

  // Calculate Sunday by adding 6 days to Monday
  const sunday = new Date(monday);
  sunday.setDate(monday.getDate() + 6);

  return [monday, sunday];
}

const SeriesSearchCard: React.FC<SearchCardProps> = ({ 
  onSearch, 
  searchValue, 
  onSave,
  onCancel,
  daySelected
}) => {
  const videoViews = 0;
  const token = useToken();
  const gym = useGetGym();
  const [series, setSeries] = useState<any>(null);
  const [isExiting, setIsExiting] = useState(false);
  const [value, onChange] = useState<Value>(getCurrentWeek(daySelected));
  // const { setMessage, setShow, setColor } = useMessagingContext();
  
  useEffect(() => {
    onChange(getCurrentWeek(daySelected));
  }, [daySelected]);
  
  // const handleDateRangeChange = (selectedDates: any): any => {
  //   if (selectedDates && selectedDates.length === 2) {
  //     const [startDate, endDate] = selectedDates;

  //     // Get the day of the week for both start and end dates (0 = Sunday, 1 = Monday, ..., 6 = Saturday)
  //     const startDay = startDate.getDay();
  //     const endDay = endDate.getDay();

  //     // Ensure the start date is a Monday (1) and the end date is a Sunday (0)
  //     if (startDay === 1 && endDay === 0) {
  //       const timeDifference = endDate.getTime() - startDate.getTime();
  //       const dayDifference = timeDifference / (1000 * 3600 * 24);

  //       if (dayDifference < 7) {
  //         onChange(selectedDates);

  //         setMessage("Selection is valid. Please search for a series to add.");
  //         setShow(true);
  //         setColor('success');
  //       } else {
  //         // Invalid range
  //         setMessage("Invalid selection. The range must be exactly one week (Monday to Sunday).");
  //         setShow(true);
  //         setColor('danger');
  //       }
  //     } else {
  //       setMessage("Invalid selection. Start date must be Monday, and end date must be Sunday.");
  //       setShow(true);
  //       setColor('danger');
  //     }
  //   } else {
  //     setMessage("Please select an entire week (Monday to Sunday).");
  //     setShow(true);
  //     setColor('danger');
  //   }
  // };

  if (gym.isPending) {
    return null;
  }

  const popover = (
    <Popover id="popover-basic">
      <Popover.Body>
        Coming Soon!
      </Popover.Body>
    </Popover>
  );

  const handleCancel = () => {
    setIsExiting(true);
    setTimeout(() => {
      onCancel && onCancel();
    }, 500);
  };

  const fetchValues = async (inputValue?: string) => {
    const { data: { data } } = await gymApi.get<any>('/gym-series', {
      params: {
        ...(inputValue && { title: inputValue }),
        ...(!inputValue && { limit: 10 }),
        gym_id: gym?.data?.id,
      },
      headers: {
        Authorization: `Bearer ${token}`,
      }
    });
  
    const seriesOptions = data?.length > 0 ? data.map((series: any) => ({
      label: series.title,
      value: series,
      type: 'series' as const,
    })) : [{ label: "No series found", value: "" }];
  
    return [
      { label: 'Series', options: seriesOptions },
    ];
  };
  
  return (
    <Row style={{ marginBottom: 20 }}>
      <div className={`${styles.cardWrapper} ${isExiting ? styles.out : ''}`}>
        <div className={styles.thumbnailContainer}>
          {series && <span className={styles.difficultyTag}>{series?.difficulties?.length > 0 && series?.difficulties[0]}</span>}
          <Image
            src={(series?.videos?.length > 0 ? series?.videos[0]?.thumbnail_url : '/placeholder.png')} 
            className={styles.thumbnail}
          />
          {series && <div className={styles?.videoCountOverlay}>{`${series?.videos?.length} ${series?.videos?.length > 1 ? 'videos' : 'video'}`}</div>}
        </div>
        <div className={styles.textContainer}>
          <div>
            <div className={styles.headerRow}>
              <Search 
                placeholder="Search for a Series"
                onChange={(searchOption: any) => {
                  setSeries(searchOption?.value);
                  onSearch(searchOption?.label);
                }}
                width={400}
                value={searchValue}
                fetchValues={fetchValues}
                instanceId='series-search'
                showGyms={false}
              />
              
            </div>
            {/* <DateRangePicker isOpen={true} onChange={handleDateRangeChange} value={value} /> */}
            {
              series && (
                <div className={styles.coachInfoRow}>
                  <Image src={series && series?.coach_avatar || '/placeholder.png'} alt="Coach Avatar" className={styles.coachAvatar} />
                  <span className={styles.coachName}>{series?.coach_name}</span>
                </div>
              )
            }
            <div className={styles.description}>
              <ExpandableText text={series?.description} maxLength={50}/>
            </div>
            <DisciplineTags disciplinesValues={series?.disciplines} />
          </div>
          <div className={styles.footerRow}>
            <OverlayTrigger trigger={['hover', 'focus']} placement="top" overlay={popover}>
              {
                series ? (
                  <div className={styles.iconsRow}>
                    <div className={styles.iconContainer}>
                      <FaEye color='#CBD5E0' />
                      <span className={styles.iconText}>{videoViews}</span>
                      <FaHeart className={styles.icon} />
                    </div>
                  </div>
                ): (
                  <div></div>
                )
              }
            </OverlayTrigger>
            <div className={styles.footerRow}>
              <button className={styles.cancelButton} onClick={handleCancel}>
                Cancel
              </button>
              {series && (
                <button className={styles.arrowButton} onClick={() => {
                  onSave && onSave.mutate({
                    gym_id: gym?.data?.id,
                    series_id: series.id,
                    title: series.title,
                    description: series.description,
                    disciplines: [],
                    display_on_week: (value as any)[0],
                  });
                  onCancel && onCancel();
                }}>
                  Save
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </Row>
  );
};

export default SeriesSearchCard;
