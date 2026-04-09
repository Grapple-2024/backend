import { colors } from "@/util/colors";
import { Button, Modal } from "react-bootstrap";

interface DeleteAccountModalProps {
  onHide: () => void;
  onDeleted: () => void;
  show: boolean;
};

const DeleteAccountModal = ({ onHide, show, onDeleted }: DeleteAccountModalProps) => {
  return (
    <>
      <Modal
        show={show}
        size="lg"
        aria-labelledby="contained-modal-title-vcenter"
        centered
      >
        <Modal.Header closeButton>
          <Modal.Title id="contained-modal-title-vcenter">
            Modal heading
          </Modal.Title>
        </Modal.Header>
        <Modal.Body>
          <h4>Delete Account</h4>
          <p>
            Are you sure you want to delete your account? This action cannot be undone.
          </p>
        </Modal.Body>
        <Modal.Footer>
          <Button style={{
            backgroundColor: 'white',
            borderColor: 'black',
            color: 'black'
          }} onClick={onHide}>Cancel</Button>
          <Button style={{
            backgroundColor: colors.primary,
            borderColor: colors.primary,
          }} onClick={onDeleted}>Delete Account</Button>
        </Modal.Footer>
      </Modal>
    </>
  );
};

export default DeleteAccountModal;