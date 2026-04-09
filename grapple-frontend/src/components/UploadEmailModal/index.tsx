import React, { useState, useCallback } from 'react';
import { Modal, Form, Button, Table, Nav, Alert } from 'react-bootstrap';
import { useDropzone } from 'react-dropzone';
import Papa from 'papaparse';
import * as XLSX from 'xlsx';
import styles from './UploadEmailModal.module.css';

interface UploadEmailsModalProps {
  show: boolean;
  onHide: () => void;
  onSubmit: (emails: string[]) => Promise<void>;
}

interface ParsedData {
  email: string;
  valid: boolean;
}

const UploadEmailModal: React.FC<UploadEmailsModalProps> = ({ show, onHide, onSubmit }) => {
  const [activeTab, setActiveTab] = useState<'file' | 'paste'>('file');
  const [textInput, setTextInput] = useState('');
  const [parsedData, setParsedData] = useState<ParsedData[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const validateEmail = (email: string): boolean => {
    const regex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return regex.test(email);
  };

  const processEmails = (rawEmails: string[]): ParsedData[] => {
    return rawEmails
      .map(email => email.trim())
      .filter(email => email.length > 0)
      .map(email => ({
        email,
        valid: validateEmail(email)
      }));
  };

  const handleFileUpload = async (file: File) => {
    setError(null);
    try {
      const fileExtension = file.name.split('.').pop()?.toLowerCase();
      let emails: string[] = [];

      if (fileExtension === 'csv') {
        const text = await file.text();
        const result = Papa.parse(text, { header: true });
        const emailColumn = result.meta.fields?.find((field: any) => 
          field.toLowerCase().includes('email')
        );
        if (emailColumn) {
          emails = result.data.map((row: any) => row[emailColumn]).filter(Boolean);
        }
      } else if (['xlsx', 'xls'].includes(fileExtension || '')) {
        const data = await file.arrayBuffer();
        const workbook = XLSX.read(data);
        const sheet = workbook.Sheets[workbook.SheetNames[0]];
        const jsonData: any = XLSX.utils.sheet_to_json(sheet);
        const emailColumn = Object.keys(jsonData[0]).find(key => 
          key.toLowerCase().includes('email')
        );
        if (emailColumn) {
          emails = jsonData.map((row: any) => row[emailColumn]).filter(Boolean);
        }
      }

      if (emails.length === 0) {
        throw new Error('No valid emails found in the file');
      }

      setParsedData(processEmails(emails));
    } catch (err) {
      setError('Error processing file. Please check the format and try again.');
      console.error(err);
    }
  };

  const handleTextInputChange = (value: string) => {
    setTextInput(value);
    const emails = value.split(/[,\n]/).map(email => email.trim());
    setParsedData(processEmails(emails));
  };

  const handleSubmit = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const validEmails = parsedData
        .filter(data => data.valid)
        .map(data => data.email);
      
      if (validEmails.length === 0) {
        throw new Error('No valid emails to submit');
      }

      await onSubmit(validEmails);
      onHide();
    } catch (err) {
      setError('Error submitting emails. Please try again.');
      console.error(err);
    } finally {
      setIsLoading(false);
    }
  };

  const onDrop = useCallback((acceptedFiles: File[]) => {
    if (acceptedFiles.length > 0) {
      handleFileUpload(acceptedFiles[0]);
    }
  }, []);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'text/csv': ['.csv'],
      'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': ['.xlsx'],
      'application/vnd.ms-excel': ['.xls']
    },
    multiple: false
  });

  return (
    <Modal show={show} onHide={onHide} size="lg">
      <Modal.Header closeButton>
        <Modal.Title>Upload Emails</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <Nav variant="tabs" className="mb-3">
          <Nav.Item>
            <Nav.Link 
              active={activeTab === 'file'}
              onClick={() => setActiveTab('file')}
            >
              Upload File
            </Nav.Link>
          </Nav.Item>
          <Nav.Item>
            <Nav.Link
              active={activeTab === 'paste'}
              onClick={() => setActiveTab('paste')}
            >
              Paste Emails
            </Nav.Link>
          </Nav.Item>
        </Nav>

        {error && (
          <Alert variant="danger" className="mb-3">
            {error}
          </Alert>
        )}

        {activeTab === 'file' ? (
          <>
            <div {...getRootProps()} className={styles.dropzone}>
              <input {...getInputProps()} />
              {isDragActive ? (
                <p>Drop the file here...</p>
              ) : (
                <div>
                  <p>Drag and drop a file here, or click to select a file</p>
                  <small className="text-muted">
                    Accepted formats: CSV, XLSX, XLS
                    <br />
                    File should contain a column with &quote;email&quote; in the header
                  </small>
                </div>
              )}
            </div>
          </>
        ) : (
          <Form.Group>
            <Form.Label>Paste emails (separated by commas or new lines)</Form.Label>
            <Form.Control
              as="textarea"
              rows={5}
              value={textInput}
              onChange={(e) => handleTextInputChange(e.target.value)}
              placeholder="example1@email.com, example2@email.com, example3@email.com"
            />
          </Form.Group>
        )}

        {parsedData.length > 0 && (
          <div className={styles.previewContainer}>
            <h6 className="mt-4 mb-3">Preview ({parsedData.length} emails)</h6>
            <Table striped bordered hover size="sm">
              <thead>
                <tr>
                  <th>Email</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                {parsedData.slice(0, 5).map((data, index) => (
                  <tr key={index} className={data.valid ? '' : 'table-danger'}>
                    <td>{data.email}</td>
                    <td>{data.valid ? 'Valid' : 'Invalid format'}</td>
                  </tr>
                ))}
                {parsedData.length > 5 && (
                  <tr>
                    <td colSpan={2} className="text-center">
                      And {parsedData.length - 5} more...
                    </td>
                  </tr>
                )}
              </tbody>
            </Table>
            <div className="text-muted">
              Valid emails: {parsedData.filter(d => d.valid).length}
            </div>
          </div>
        )}
      </Modal.Body>
      <Modal.Footer>
        <Button className={styles.buttonDanger} onClick={onHide}>
          Cancel
        </Button>
        <Button
          onClick={handleSubmit}
          className={styles.button}
          disabled={isLoading || parsedData.filter(d => d.valid).length === 0}
        >
          {isLoading ? 'Uploading...' : 'Upload'}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default UploadEmailModal;