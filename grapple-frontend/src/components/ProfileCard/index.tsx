import Avatar from "@/components/Avatar";
import { useMobileContext } from "@/context/mobile";
import { colors } from "@/util/colors";
import { useState } from "react";
import { Button, Col, Row } from "react-bootstrap";
import EditPhotoModal from "../EditPhotoModal/EditPhotoModal";
import { useUpdateAvatar } from "@/hook/profile";


interface ProfileCardProps {
  src: string;
  firstName: string;
  title: string;
};

const ProfileCard = ({ src, firstName, title }: ProfileCardProps) => {
  const { isMobile } = useMobileContext();
  const [isChangingPhoto, setIsChangingPhoto] = useState(false);
  const updateAvatar = useUpdateAvatar();
  
  return (
    <div style={{
      borderRadius: 10, 
      backgroundColor: 'white',
      padding: 20,
    }}>
      <Row style={{
        padding: isMobile ? 10 : 0,
      }}>
        <Col xs={isMobile ? 7 : 2} style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
        }}>
          <Avatar height={100} src={src} />
        </Col>
        <Col style={{
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          alignItems: 'flex-start'
        }}>
          <Row xs={isMobile ? 12 : 8}>
            <h2>{firstName}</h2>
          </Row>
          <Row xs={isMobile ? 12 : 4}>
            <h5>{title}</h5>
          </Row>      
        </Col>
        <Col xs={isMobile ? 12 : 2} style={{
          display: 'flex',
          justifyContent: isMobile ? 'flex-start' : 'center',
          alignItems: 'center',
        }}>
          <Button style={{
            backgroundColor: colors.black,
            borderColor: colors.black,
          }} onClick={() => setIsChangingPhoto(true)}>
            Change Photo
          </Button>
        </Col>
      </Row>
      <EditPhotoModal 
        isOpen={isChangingPhoto} 
        imageType="avatar"
        onClose={() => setIsChangingPhoto(false)}
        onSave={(newPhoto: any, name: string) => {
          updateAvatar.mutate({ 
            file: newPhoto, 
            name: name 
          });
        }}
      />
    </div>
  );
};

export default ProfileCard;
