'use client';

import styles from './GymProfile.module.css';
import { useState } from "react";
import PageSkeleton from "@/components/PageSkeleton";
import ProfileSchedule from '@/components/Profile/Display/Schedule';
import { Col, Modal, Row } from 'react-bootstrap';
import DatePicker from '@/components/DatePicker';
import EditPhotoModal from '@/components/EditPhotoModal/EditPhotoModal';
import Scheduler from '../create-gym/components/Scheduler';
import ImageSection from '@/components/Profile/ImageSection';
import LogoSection from '@/components/Profile/LogoSection';
import DescriptionSection from '@/components/Profile/DescriptionSection';
import HeroSection from '@/components/Profile/HeroSection';
import DisciplinesSection from '@/components/Profile/DisciplinesSection';
import ConfirmationModal from '@/components/ConfirmationModal';
import { useDeleteGym, useGetGym, useUpdateGym, useUploadGymImage } from '@/hook/gym';
import { Gym, GymSchedule } from '@/types/gym';

const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];


interface EditScheduleModalProps {
  isOpen: boolean;
  schedule: any;
  onClose: () => void;
  onSave: (newSchedule: any) => void;
}

const EditScheduleModal = ({ isOpen, onClose, onSave, schedule }: EditScheduleModalProps) => {
  const [formData, setFormData] = useState<any>(schedule);
  
  return (
    <Modal 
      show={isOpen} 
      onHide={onClose} 
      centered
      contentClassName={styles.modalContent}
    >
      <Modal.Header 
        closeButton 
        className={styles.modalHeader}
        closeVariant="dark"
      >
        <Modal.Title className={styles.modalTitle}>Edit Schedule</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <Scheduler schedule={formData} setSchedule={setFormData}/>
      </Modal.Body>
      <Modal.Footer className={styles.modalFooter}>
        <button 
          className={styles.secondaryButton}
          onClick={onClose}
        >
          Cancel
        </button>
        <button
          className={styles.primaryButton}
          onClick={() => onSave(formData)}
        >
          Save Changes
        </button>
      </Modal.Footer>
    </Modal>
  );
};

const GymProfile = () => {
  const gymData = useGetGym();
  const updateGym = useUpdateGym();
  const updateGymImage = useUploadGymImage();
  
  const [daySelected, setDaySelected] = useState<Date>(new Date());
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingImage, setEditingImage] = useState<'banner' | 'logo' | 'hero' | null>(null);
  const [showConfirmationModal, setShowConfirmationModal] = useState(false);
  const deleteGym = useDeleteGym();

  const [isEditingSchedule, setIsEditingSchedule] = useState(false);

  const gym = gymData?.data;
  
  function getCurrentDay(daySelected: Date): string {
    const daysOfWeek = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    const currentDayIndex = daySelected.getDay();
    return daysOfWeek[currentDayIndex - 1 === -1 ? 6 : currentDayIndex - 1];
  }

  if (gymData?.isPending) {
    return <PageSkeleton />;
  }

  return (
    <div className={styles.container}>
      <Row style={{ margin: 0, padding: 0 }}>
        <Col xs={12} md={8} style={{ backgroundColor: 'white', padding: 0 }}>
          <ImageSection 
            isCoach
            bannerUrl={gym?.banner_url || '/placeholder-banner.png'}
            setEditingImage={(imageType: string) => setEditingImage(imageType as any)}
            setIsModalOpen={(isOpen: boolean) => setIsModalOpen(isOpen)}
          />

          <div className={styles.contentGrid}>
            <div className={styles.mainContent}>
              <div className={styles.gymInfo}>
                <LogoSection 
                  isCoach
                  logoUrl={gym?.logo_url || '/placeholder-logo.jpeg'}
                  setEditingImage={(imageType: string) => setEditingImage(imageType as any)}
                  setIsModalOpen={(isOpen: boolean) => setIsModalOpen(isOpen)}
                  updateGym={updateGym}
                  gym={gym}
                />
                <div className={styles.buttonGroup}>
                  <button className={styles.primaryButton} onClick={() => setShowConfirmationModal(true)}>
                    Delete this gym
                  </button>
                </div>
              </div>

              <section className={styles.section}>
                <DescriptionSection 
                  isCoach
                  gym={gym}
                  updateGym={updateGym}
                />
              </section>

              <section className={styles.videoSection}>
                <HeroSection 
                  isCoach
                  gym={gym}
                  setEditingImage={setEditingImage}
                  setIsModalOpen={setIsModalOpen}
                />
              </section>

              <section className={styles.section}>
                <DisciplinesSection 
                  isCoach
                  gym={gym}
                  updateGym={updateGym}
                />
              </section>
            </div>
          </div>
        </Col>
        
        <Col xs={12} md={4} style={{ margin: 0, height: '92vh' }}>
          <Row style={{ margin: 0, paddingRight: 10, paddingTop: 10, height: '16vh' }}>
            <DatePicker 
              onDaySelect={(day) => {
                setDaySelected(day);
              }}
              isProfilePage
            />
          </Row>
          <Row style={{ margin: 0, paddingRight: 10, paddingTop: 30, height: '70vh' }}>
            <ProfileSchedule 
              schedule={gym?.schedule as GymSchedule}
              days={days}
              selectedDay={getCurrentDay(daySelected)}
              daily
            />
          </Row>
          <Row>
            <div className={styles.footerRow}>
              <button className={styles.saveButton} onClick={() => {
                setIsEditingSchedule(true);
              }}>
                Edit Schedule
              </button>
            </div>
          </Row>
        </Col>
      </Row>
      <EditPhotoModal
        isOpen={isModalOpen}
        onClose={() => {
          setIsModalOpen(false);
          setEditingImage(null);
        }}
        imageType={editingImage as any} // 'banner' | 'logo' | 'hero'
        onSave={(newPhoto: any, name: string) => {
          if (editingImage === 'banner') {
            updateGymImage.mutate({
              file: newPhoto,
              fileType: 'banner',
              name,
            });
          } else if (editingImage === 'logo') {
            updateGymImage.mutate({
              file: newPhoto,
              fileType: 'logo',
              name,
            });
          } else if (editingImage === 'hero') {
            updateGymImage.mutate({
              file: newPhoto,
              fileType: 'hero',
              name,
            });
          }
        }}
      />
      <EditScheduleModal 
        isOpen={isEditingSchedule}
        schedule={gym?.schedule}
        onClose={() => setIsEditingSchedule(false)}
        onSave={(newSchedule: any) => {
          updateGym.mutate({
            ...gym,
            schedule: newSchedule,
          } as any)
          setIsEditingSchedule(false);
        }}
      />
      <ConfirmationModal 
        show={showConfirmationModal}
        setShow={setShowConfirmationModal}
        onConfirm={() => deleteGym.mutate()}
      />
    </div>
  );
};

export default GymProfile;