// QRCodeGenerator.tsx
import React, { useState } from 'react';
import { QRCodeCanvas } from 'qrcode.react';
import { Modal, Button } from 'react-bootstrap';
import styles from './QrCode.module.css';

interface QRCodeGeneratorProps {
  defaultValue?: string;
  size?: number;
}

const QRCodeGenerator: React.FC<QRCodeGeneratorProps> = ({
  defaultValue = '',
  size = 256
}) => {
  const [showModal, setShowModal] = useState(false);
  const [qrValue] = useState(defaultValue);

  const handleDownload = () => {
    const canvas = document.querySelector('canvas');
    if (canvas) {
      const url = canvas.toDataURL('image/png');
      const link = document.createElement('a');
      link.href = url;
      link.download = 'qrcode.png';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      setShowModal(false);
    }
  };

  const handlePrint = () => {
    const canvas = document.querySelector('canvas');
    if (canvas) {
      const url = canvas.toDataURL('image/png');
      const windowContent = `
        <html>
          <body style="display:flex;justify-content:center;align-items:center;height:100vh;margin:0">
            <img src="${url}"/>
          </body>
        </html>
      `;
      const printWindow = window.open('', '', 'height=600,width=800');
      if (printWindow) {
        printWindow.document.write(windowContent);
        printWindow.document.close();
        printWindow.focus();
        printWindow.print();
        printWindow.close();
        setShowModal(false);
      }
    }
  };

  return (
    <>
      <Button 
        className={styles.mainButton} 
        onClick={() => setShowModal(true)}
              variant="dark"
      >
        Your QR Code
      </Button>

      <Modal 
        show={showModal} 
        onHide={() => setShowModal(false)}
        centered
        size="sm"
        className={styles.modal}
      >
        <Modal.Header closeButton>
        <h6 className={styles.title}>Gym Sign Up URL</h6>
        </Modal.Header>
        <Modal.Body className={styles.modalBody}>
          <div className={styles.modalQrContainer}>
            <QRCodeCanvas
              value={qrValue || ' '}
              size={size}
              level="H"
              includeMargin
              className={styles.qrCode}
            />
          </div>
          <div className={styles.actions}>
            <Button
              variant="dark"
              onClick={handleDownload}
              className={styles.button}
              disabled={!qrValue}
            >
              Download
            </Button>
            <Button
              variant="dark"
              onClick={handlePrint}
              className={styles.button}
              disabled={!qrValue}
            >
              Print
            </Button>
          </div>
        </Modal.Body>
      </Modal>
    </>
  );
};

export default QRCodeGenerator;