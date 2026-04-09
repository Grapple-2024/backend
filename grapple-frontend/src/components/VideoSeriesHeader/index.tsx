import { useContentContext } from "@/context/content";
import { useEditSeriesContext } from "@/context/edit-series";
import { useMobileContext } from "@/context/mobile";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Col, Image, Row } from "react-bootstrap";
import { FaRegEdit } from "react-icons/fa";

const VideoSeriesHeader = ({
  title,
  seriesDescription,
  isCoach = false
}: any) => {
  const [isHovering , setIsHovering] = useState(false);
  const router = useRouter();
  const { isMobile } = useMobileContext();
  const { setIsEditing, setCurrentSeries, isEditing, setStep, setSeries } = useEditSeriesContext();
  const { setOpen } = useContentContext();
  
  return (
    <Row style={{ 
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      height: '8vh',
    }}>
      <Col xs={isMobile ? 2 : 1}>
        <Image 
          onMouseEnter={() => setIsHovering(true)}
          onMouseLeave={() => setIsHovering(false)}
          src={isHovering ? '/back-button-hover.svg' : '/back-button.svg'}
          style={{
            cursor: 'pointer',
          }}
          onClick={() => {
            setIsEditing(false);
            setCurrentSeries(null);
            router.back();
          }}
          alt="Back Button" 
        />
      </Col>
      <Col style={{
        display: 'flex',  
        alignItems: 'center' 
      }}>
        <h2 style={{ margin: 0 }}>{title}</h2>
      </Col>
      {
        isCoach && (
          <Col style={{
            display: 'flex',  
            alignItems: 'center',
            justifyContent: 'flex-end',
            cursor: "pointer"
          }}> 
            <FaRegEdit size={25} onClick={() => {
              setIsEditing(true);
              setSeries({
                title,
                description: seriesDescription
              })
              
              setStep(1);
              setOpen(true);
            }} />
          </Col>
        )
      }
    </Row>
  );
}

export default VideoSeriesHeader;