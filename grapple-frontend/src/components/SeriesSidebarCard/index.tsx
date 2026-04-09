import { Dropdown, Image } from "react-bootstrap";
import styles from './SeriesSidebarCard.module.css';
import { HiOutlineDotsHorizontal } from "react-icons/hi";
import { forwardRef, useState } from "react";
import ConfirmationModal from "../ConfirmationModal";


const CustomToggle = forwardRef<HTMLAnchorElement, { onClick: (e: React.MouseEvent<HTMLAnchorElement>) => void }>(
  ({ onClick }, ref) => (
    <a
      href=""
      ref={ref}
      onClick={(e) => {
        e.preventDefault();
        onClick(e);
      }}
      style={{ color: 'inherit', textDecoration: 'none' }}
    >
      <HiOutlineDotsHorizontal size={30} />
    </a>
  )
);

interface SeriesSidebarCardProps {
  currentSelection: any;
  setCurrentSelection: (video: any) => void;
  video: any;
  durations: string[];
  index: number;
  handleEdit: (index: number) => void;
  handleDelete: (id: string) => void;
  coachAvatar: string;
  coachName: string;
  isCoach?: boolean;
};

const SeriesSidebarCard = ({
  currentSelection,
  setCurrentSelection,
  video,
  durations,
  index,
  handleEdit,
  handleDelete,
  coachAvatar,
  coachName,
  isCoach = false
}: SeriesSidebarCardProps) => {
  const [show, setShow] = useState(false);

  return (
    <>
      <div
        className={`${styles.videoItem} ${currentSelection?.id === video.id ? styles.selected : ""}`}
        onClick={() => setCurrentSelection(video)} 
      >
        <div className={styles.thumbnailWrapper}>
          <Image
            src={video?.thumbnail_url || "https://via.placeholder.com/150"}
            className={styles.thumbnail}
          />
          <span className={styles.duration}>
            {durations[index] !== "Error" ? durations[index] : "N/A"}
          </span>
        </div>
        <div className={styles.videoInfo}>
          <div style={{
            margin: 0,
            display: 'flex',
            marginTop: -25,
            justifyContent: 'flex-end',
          }}>
            {
              isCoach && (
                <Dropdown>
                  <Dropdown.Toggle as={CustomToggle} id={`dropdown-${video.id}`} />
                  <Dropdown.Menu>
                    <Dropdown.Item onClick={(e) => {
                      handleEdit(index);
                    }}>
                      Edit
                    </Dropdown.Item>
                    <Dropdown.Divider />
                    <Dropdown.Item onClick={(e) => {
                      setShow(true);
                    }}>
                      Delete
                    </Dropdown.Item>
                  </Dropdown.Menu>
                </Dropdown>
              )
            }
          </div>
          <h3 className={styles.title}>{video.title}</h3>
          <h6 className={styles.subTitle}>{video.difficulty}</h6>
          <div className={styles.coachInfo}>
            <Image
              src={coachAvatar || "https://via.placeholder.com/50"}
              alt="Coach Avatar"
              className={styles.coachAvatar}
            />
            <span className={styles.coachName}>{coachName}</span>
          </div>
        </div>
      </div>

      <ConfirmationModal 
        show={show}
        setShow={setShow}
        onConfirm={() => {
          handleDelete(video.id);
          setShow(false);
        }}
      />
    </>
  )
};

export default SeriesSidebarCard;