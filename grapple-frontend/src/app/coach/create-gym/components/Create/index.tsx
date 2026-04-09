import TimePicker from "@/components/TimePicker";
import { useMessagingContext } from "@/context/message";
import { useEffect, useState } from "react";
import { Button, ButtonGroup, Col, Dropdown, Form, InputGroup, Row, ToggleButton } from "react-bootstrap";
import { FaPlus, FaPlusCircle } from "react-icons/fa";

interface CreateProps {
  setSchedule: any;
  schedule: any;
  day: string;
};

const times = [
  '12:00',
  '12:15',
  '12:30',
  '12:45',
  '1:00',
  '1:15',
  '1:30',
  '1:45',
  '2:00',
  '2:15',
  '2:30',
  '2:45',
  '3:00',
  '3:15',
  '3:30',
  '3:45',
  '4:00',
  '4:15',
  '4:30',
  '4:45',
  '5:00',
  '5:15',
  '5:30',
  '5:45',
  '6:00',
  '6:15',
  '6:30',
  '6:45',
  '7:00',
  '7:15',
  '7:30',
  '7:45',
  '8:00',
  '8:15',
  '8:30',
  '8:45',
  '9:00',
  '9:15',
  '9:30',
  '9:45',
  '10:00',
  '10:15',
  '10:30',
  '10:45',
  '11:00',
  '11:15',
  '11:30',
  '11:45',
];

const Create = ({ setSchedule, schedule, day }: CreateProps) => {
  const {
    setMessage,
    setShow,
    setColor,
  } = useMessagingContext();

  const [formData, setFormData] = useState({
    title: '',
    start: '',
    end: '',
    pm: true,
  });
  
  return (
    <>
      <Form style={{ marginBottom: 10 }}>
        <Row>
          <Form.Group className="mb-3" controlId="exampleForm.ControlInput1">
            <Form.Label>Class Title</Form.Label>
            <Form.Control 
              type="text" 
              placeholder="Enter a class name" 
              value={formData.title}
              onChange={e => setFormData({ ...formData, 'title': e.target.value })}
            />
          </Form.Group>
        </Row>
        <Row>
          <Col>
            <TimePicker 
              start="05:00" 
              end="24:00"
              label="Start Time"
              step={15} 
              onChange={(e: any) => {
                setFormData({
                  ...formData,
                  start: e,
                });
              }}
            />
          </Col>
          <Col>
            <TimePicker 
              start="05:00" 
              end="24:00"
              label="End Time"
              step={15} 
              onChange={(e: any) => {
                setFormData({
                  ...formData,
                  end: e,
                });
              }}
            />
          </Col>
        </Row>
        <Row style={{ marginTop: 30 }}>
          <Col xs={8}></Col>
          <Col xs={4} style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button variant="dark" style={{ 
              display: 'flex', 
              alignItems: 'center' 
            }} 
            onClick={(e) => {
              e.preventDefault();

              if (formData.start === '' || formData.end === '') {
                setMessage('Please select a start and end time');
                setColor('danger');
                setShow(true);
              } else if (formData.title === '') {
                setMessage('Please enter a class title');
                setColor('danger');
                setShow(true);
              } else {
                setSchedule({
                  ...schedule,
                  [day.toLowerCase()]: [
                    formData,
                    ...(schedule[day.toLowerCase()] || []),
                  ],
                });
  
                setFormData({
                  title: '',
                  start: '',
                  end: '',
                  pm: formData.pm,
                });

              };
            }}
            >
              <FaPlusCircle style={{ marginRight: 10}}/> 
              Add
            </Button>
          </Col>
        </Row>
      </Form>
    </>
  );
};

export default Create;