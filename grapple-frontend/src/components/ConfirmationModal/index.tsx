import { Button, Modal } from "react-bootstrap";
import styles from "./ConfirmationModal.module.css";

const ConfirmationModal = ({
  show,
  setShow,
  onConfirm,
  children = null,
}: any) => {
  return (
    <div
      className="modal show"
      style={{ display: show ? 'block': 'none', position: 'initial' }}
    >
    <Modal show={show} onHide={() => setShow(false)}>
      <Modal.Header closeButton>
        <Modal.Title>Confirm decision</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <p>{children ? children : 'Are you sure you want to delete this?'}</p>
      </Modal.Body>
      <Modal.Footer>
        <Button
          variant="dark"
          className={styles.button}
          onClick={() => setShow(false)}
        >
          Cancel
        </Button>
        <Button
          variant="dark"
          className={styles.buttonDanger}
          onClick={onConfirm}
        >
          Confirm
        </Button>
      </Modal.Footer>
    </Modal>
  </div>
  );
};

export default ConfirmationModal;