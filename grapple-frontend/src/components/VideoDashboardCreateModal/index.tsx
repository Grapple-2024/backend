import React, { useEffect, useState } from 'react';
import { Modal, Button, Form, Row, Col, Dropdown, Image, Spinner } from 'react-bootstrap';
import styles from './VideoDashboardCreateModal.module.css';
import { useContentContext } from '@/context/content';
import { difficulty, disciplines } from '@/util/default-values';
import { CiFilter } from 'react-icons/ci';
import { useEditSeriesContext } from '@/context/edit-series';
import { useLoadingContext } from '@/context/loading';
import { useCreateSeries, useDeleteSeries, useUpdateSeries } from '@/hook/series';
import { useGetGym } from '@/hook/gym';
import { useAddVideo, useCreateVideo, useUpdateVideo } from '@/hook/video';
import { useGetUserProfile } from '@/hook/profile';

const VideoModalDropdown = ({ 
  title, 
  data, 
  values,
  setValues,
  isMulti = true,
}: any) => {
  const [selectedItems, setSelectedItems] = useState<any>([]);
  
  const toggleItemSelection = (item: string) => {
    setSelectedItems((prevSelected: any) => {
      if (prevSelected.includes(item)) {
        return prevSelected.filter((i: any) => i !== item); // Deselect item
      } else {
        if (isMulti) {
          return [...prevSelected, item];
        } else {
          return [item];
        }
      }
    });
  };

  useEffect(() => {
    setSelectedItems(values);
  }, []);  

  useEffect(() => {
    setValues(selectedItems);
  }, [selectedItems]);

  const selectedText = selectedItems?.length > 0 ? selectedItems?.join(', ') : `Choose ${title}`;

  if (!selectedItems) {
    return null;
  }

  return (
    <Form.Group className={styles.formGroup} controlId={title}>
      <Dropdown>
        <Dropdown.Toggle variant="light" id="dropdown-basic" className={styles.dropdownToggle}>
          <div className={styles.dropdownToggleIcon}>
            <CiFilter />
            <span>{selectedText}</span>
          </div>
        </Dropdown.Toggle>
        <Dropdown.Menu className={styles.dropdownMenu}>
          {data.map((item: any, index: number) => (
            <Dropdown.Item
              key={index}
              className={styles.dropdownItem}
              onClick={() => toggleItemSelection(item)}
              active={selectedItems.includes(item)}
            >
              {selectedItems.includes(item) ? `✅ ${item}` : item}
            </Dropdown.Item>
          ))}
        </Dropdown.Menu>
      </Dropdown>
    </Form.Group>
  );
};

const isIOS = (): boolean => {
  return [
    'iPad Simulator',
    'iPhone Simulator',
    'iPod Simulator',
    'iPad',
    'iPhone',
    'iPod'
  ].includes(navigator.platform)
  || (navigator.userAgent.includes("Mac") && "ontouchend" in document);
};

const isSafari = (): boolean => {
  return /^((?!chrome|android).)*safari/i.test(navigator.userAgent);
};

const validateFrame = (canvas: HTMLCanvasElement): boolean => {
  const ctx = canvas.getContext('2d');
  if (!ctx) return false;

  // Get the image data
  const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
  const pixels = imageData.data;

  // Check if the frame is all white or all black
  let isBlank = true;
  let lastPixel = pixels[0];

  // Sample pixels (check every 100th pixel for performance)
  for (let i = 0; i < pixels.length; i += 400) {
    if (pixels[i] !== lastPixel || pixels[i + 1] !== lastPixel || pixels[i + 2] !== lastPixel) {
      isBlank = false;
      break;
    }
  }

  return !isBlank;
};

const generateThumbnail = async (videoFile: File): Promise<{ url: string; file: File }> => {
  return new Promise((resolve, reject) => {
    const video = document.createElement('video');
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');

    let attemptCount = 0;
    const maxAttempts = 5;
    const timePoints = [3, 2, 1, 0.5, 4]; // Different time points to try

    // Set up video element
    video.preload = 'metadata';
    video.playsInline = true;
    video.muted = true;
    video.crossOrigin = 'anonymous';

    if (isIOS() || isSafari()) {
      video.autoplay = true;
    }

    const objectUrl = URL.createObjectURL(videoFile);
    video.src = objectUrl;

    const cleanup = () => {
      video.pause();
      video.removeAttribute('src');
      video.load();
      URL.revokeObjectURL(objectUrl);
    };

    const generateFrame = async () => {
      return new Promise<void>((frameResolve, frameReject) => {
        const tryCapture = () => {
          if (attemptCount >= maxAttempts) {
            frameReject(new Error('Failed to generate valid thumbnail after multiple attempts'));
            return;
          }

          try {
            // Set canvas dimensions
            canvas.width = video.videoWidth;
            canvas.height = video.videoHeight;

            // Draw the current frame
            ctx?.drawImage(video, 0, 0, canvas.width, canvas.height);

            // Validate the frame
            if (validateFrame(canvas)) {
              canvas.toBlob(
                (blob) => {
                  if (!blob) {
                    frameReject(new Error('Failed to generate thumbnail blob'));
                    return;
                  }

                  const thumbnailFileName = videoFile.name.replace(/\.[^/.]+$/, '') + '_thumbnail.png';
                  const file = new File([blob], thumbnailFileName, { type: 'image/png' });
                  const url = URL.createObjectURL(blob);

                  cleanup();
                  frameResolve();
                  resolve({ url, file });
                },
                'image/png',
                1.0
              );
            } else {
              // Try next time point
              attemptCount++;
              video.currentTime = timePoints[attemptCount % timePoints.length];
            }
          } catch (error) {
            frameReject(error);
          }
        };

        // For iOS, we need to wait a bit after seeking
        const captureWithDelay = () => {
          setTimeout(tryCapture, 100);
        };

        if (isIOS() || isSafari()) {
          captureWithDelay();
        } else {
          tryCapture();
        }
      });
    };

    // Set up event handlers
    video.onloadedmetadata = () => {
      if (video.duration < timePoints[0]) {
        // If video is shorter than our first time point, use the middle
        video.currentTime = video.duration / 2;
      } else {
        video.currentTime = timePoints[0];
      }
    };

    video.onseeked = async () => {
      try {
        await generateFrame();
      } catch (error) {
        // If frame generation fails, try the next time point
        attemptCount++;
        if (attemptCount < maxAttempts) {
          video.currentTime = timePoints[attemptCount % timePoints.length];
        } else {
          cleanup();
          reject(error);
        }
      }
    };

    // Error handling
    video.onerror = () => {
      cleanup();
      reject(new Error('Error loading video'));
    };

    // Set up timeout
    const timeout = setTimeout(() => {
      cleanup();
      reject(new Error('Thumbnail generation timed out'));
    }, 20000); // 20 second timeout

    // Clear timeout if successful
    video.oncanplay = () => {
      clearTimeout(timeout);
    };

    // For iOS, we might need to play the video briefly
    if (isIOS() || isSafari()) {
      video.oncanplay = async () => {
        clearTimeout(timeout);
        try {
          await video.play();
          setTimeout(() => {
            video.pause();
            video.currentTime = timePoints[0];
          }, 100);
        } catch (error) {
          // If autoplay fails, try without it
          video.currentTime = timePoints[0];
        }
      };
    }
  });
};

// Usage example with retry logic
const createThumbnail = async (videoFile: File): Promise<{ url: string; file: File }> => {
  let attempts = 0;
  const maxAttempts = 3;

  while (attempts < maxAttempts) {
    try {
      return await generateThumbnail(videoFile);
    } catch (error) {
      attempts++;
      if (attempts === maxAttempts) {
        throw error;
      }
      // Wait before retrying
      await new Promise(resolve => setTimeout(resolve, 1000));
    }
  }

  throw new Error('Failed to generate thumbnail after all attempts');
};

const FileUpload = ({ setFile, setThumbnail }: any) => {
  const [fileName, setFileName] = useState('No File Selected');

  const handleFileChange = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (file) {
      setFileName(file.name);
      setFile(file);
      
      try {
        const thumbnailFile = await createThumbnail(file);
        setThumbnail(thumbnailFile);
      } catch (error) {
        console.error('Error generating thumbnail:', error);
      }
    } else {
      setFileName('No File Selected');
      setFile(null);
      setThumbnail('');
    }
  };

  return (
    <div className={styles.fileUploadContainer}>
      <label htmlFor="formFile" className={styles.uploadButton}>
        <Image src="/uploader.svg" alt="Upload" />
      </label>
      <input 
        type="file" 
        id="formFile" 
        onChange={handleFileChange} 
        className={styles.hiddenInput}
      />
      <span className={styles.fileName}>{fileName}</span>
    </div>
  );
};

interface ValidationErrors {
  title?: string;
  description?: string;
  difficulty?: string;
  disciplines?: string;
  file?: string;
};

const VideoDashboardCreateModal = () => {
  const { 
    currentSeries, 
    series,
    setSeries,
    setFormData,
    formData, 
    isAdding,
    setIsAdding
  } = useEditSeriesContext();
  const { open, setOpen } = useContentContext();
  const { isEditing, setIsEditing, step, setStep } = useEditSeriesContext();
  const [file, setFile] = useState<File | null>(null);
  const [thumbnail, setThumbnail] = useState<string>('');
  const [errors, setErrors] = useState<ValidationErrors>({});
  const gym = useGetGym();
  const createSeries = useCreateSeries();
  const deleteSeries = useDeleteSeries();
  const createVideo = useCreateVideo();
  const updateSeries = useUpdateSeries(currentSeries?.id);
  const updateVideo = useUpdateVideo();
  const addVideo = useAddVideo();
  const profile = useGetUserProfile();
  const { loading } = useLoadingContext();
  const coachName = profile?.data?.first_name + ' ' + profile?.data?.last_name;
  
  const validateSeriesForm = (): boolean => {
    const newErrors: ValidationErrors = {};
    let isValid = true;

    if (!series.title?.trim()) {
      newErrors.title = 'Series title is required';
      isValid = false;
    } else if (series.title.length > 100) {
      newErrors.title = 'Series title must be less than 100 characters';
      isValid = false;
    }
    
    setErrors(newErrors);
    return isValid;
  };

  const validateVideoForm = (): boolean => {
    const newErrors: ValidationErrors = {};
    let isValid = true;

    if (!formData?.title?.trim()) {
      newErrors.title = 'Video title is required';
      isValid = false;
    } else if (formData.title.length > 100) {
      newErrors.title = 'Video title must be less than 100 characters';
      isValid = false;
    }

    if (!formData?.description?.trim()) {
      newErrors.description = 'Video description is required';
      isValid = false;
    } else if (formData.description.length > 500) {
      newErrors.description = 'Video description must be less than 500 characters';
      isValid = false;
    }

    if (!formData?.difficulty) {
      newErrors.difficulty = 'Please select a difficulty level';
      isValid = false;
    }

    if (!formData?.disciplines?.length) {
      newErrors.disciplines = 'Please select at least one discipline';
      isValid = false;
    }

    if (!isEditing && !file) {
      newErrors.file = 'Please upload a video file';
      isValid = false;
    }

    setErrors(newErrors);
    return isValid;
  };

  const handleCancel = () => {
    setOpen(false);
    setErrors({});
    
    if (currentSeries && !isAdding) {
      deleteSeries.mutate(currentSeries?.id);
      setIsAdding(false);
      setFormData({
        title: '',
        description: '',
        presigned_url: '',
        difficulty: '',
        disciplines: [],
      });
    }

    setStep(step - 1);
  };

  useEffect(() => {
    if (!loading) {
      setOpen(false);
    }
  }, [loading]);
  
  const handleContinue = () => {
    if (step === 1) {
      if (!validateSeriesForm()) return;

      if (isEditing) {
        updateSeries.mutate({
          ...currentSeries,
          title: series.title,
          description: series.description == '' ? "Empty" : series.description,
        });

        setIsEditing(false);
        setOpen(false);
      } else {
        createSeries.mutate({
          ...series,
          description: series.description == '' ? "Empty" : series.description,
          gym_id: gym?.data?.id,
          videos: [],
          coach_name: coachName,
          coach_avatar: profile?.data?.avatar_url,
        });
      }
      setStep(step + 1);
      setSeries({
        title: '',
        description: '',
      });
      setErrors({});
    } else {
      if (!validateVideoForm()) return;

      if (isEditing) {
        updateVideo.mutate({
          ...formData,
          series_id: currentSeries.id
        });

        setOpen(false);
        setIsEditing(false);
      } else if (isAdding) {
        addVideo.mutate({
          file, 
          thumbnail,
          seriesId: currentSeries.id,
          video: {
            ...formData,
            seriesId: currentSeries.id
          },
        });
      } else {
        createVideo.mutate({
          file, 
          thumbnail,
          seriesId: currentSeries.id,
          video: {
            ...formData,
            series_id: currentSeries.id
          },
        });
      }
      setSeries({
        title: '',
        description: '',
      });
      setFormData({
        title: '',
        description: '',
        presigned_url: '',
        difficulty: '',
        disciplines: [],
      });
      setErrors({});
    }
  };

  if (gym?.isPending || profile?.isPending) {
    return null;
  }
  
  return (
    <Modal show={open} onHide={handleCancel} centered size='lg'>
      <Modal.Header closeButton>
        <Modal.Title>{step === 1 ? isEditing ? 'Update Series': 'Create Series' : isEditing ? 'Edit Video' : 'Create Video'}</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        {step === 1 && (
          <Form noValidate>
            <Form.Group className={styles.formGroup} controlId="seriesTitle">
              <Form.Label className={styles.floatingLabel}>Series Title</Form.Label>
              <Form.Control 
                type="text" 
                placeholder="Title" 
                className={`${styles.input} ${errors.title ? 'is-invalid' : ''}`}
                value={series.title} 
                onChange={(e) => {
                  setSeries({ ...series, title: e.target.value });
                  if (errors.title) setErrors({ ...errors, title: undefined });
                }}
                isInvalid={!!errors.title}
              />
              <Form.Control.Feedback type="invalid">{errors.title}</Form.Control.Feedback>
            </Form.Group>
            <Form.Group className={styles.formGroup} controlId="seriesDescription">
              <Form.Label className={styles.floatingLabel}>Description (Optional)</Form.Label>
              <Form.Control 
                as="textarea" 
                rows={3} 
                placeholder="Description..." 
                className={`${styles.textarea} ${errors.description ? 'is-invalid' : ''}`}
                value={series.description}
                onChange={(e) => {
                  setSeries({ ...series, description: e.target.value });
                  if (errors.description) setErrors({ ...errors, description: undefined });
                }}
                isInvalid={!!errors.description}
              />
              <Form.Control.Feedback type="invalid">{errors.description}</Form.Control.Feedback>
            </Form.Group>
          </Form>
        )}

        {step === 2 && (
          <Form noValidate>
            <Row>
              <Col>
                <Form.Group className={styles.formGroup} controlId="videoTitle">
                  <Form.Label className={styles.floatingLabel}>Title</Form.Label>
                  <Form.Control 
                    type="text" 
                    placeholder="Title" 
                    className={`${styles.input} ${errors.title ? 'is-invalid' : ''}`}
                    value={formData?.title || ""}
                    onChange={(e) => {
                      setFormData({ ...formData, title: e.target.value });
                      if (errors.title) setErrors({ ...errors, title: undefined });
                    }}
                    isInvalid={!!errors.title}
                  />
                  <Form.Control.Feedback type="invalid">{errors.title}</Form.Control.Feedback>
                </Form.Group>
              </Col>
            </Row>
            <Row>
              <Col>
                <Form.Group className={styles.formGroup} controlId="videoDescription">
                  <Form.Label className={styles.floatingLabel}>Description</Form.Label>
                  <Form.Control 
                    as="textarea" 
                    rows={3} 
                    placeholder="Description..." 
                    className={`${styles.textarea} ${errors.description ? 'is-invalid' : ''}`}
                    value={formData?.description || ""}
                    onChange={(e) => {
                      setFormData({ ...formData, description: e.target.value });
                      if (errors.description) setErrors({ ...errors, description: undefined });
                    }}
                    isInvalid={!!errors.description}
                  />
                  <Form.Control.Feedback type="invalid">{errors.description}</Form.Control.Feedback>
                </Form.Group>
              </Col>
            </Row>
            <Row>
              <Col>
                <VideoModalDropdown 
                  title="Difficulty" 
                  data={difficulty.map((diff) => diff.label)} 
                  values={formData?.difficulty ? [formData.difficulty] : []}
                  setValues={(values: any) => {
                    setFormData({ ...formData, difficulty: values[0] });
                    if (errors.difficulty) setErrors({ ...errors, difficulty: undefined });
                  }}
                  isMulti={false}
                  error={errors.difficulty}
                />
              </Col>
              <Col>
                <VideoModalDropdown 
                  title="Discipline" 
                  data={disciplines.map((disc) => disc.label)} 
                  values={formData?.disciplines || []}
                  setValues={(values: any) => {
                    setFormData({ ...formData, disciplines: values });
                    if (errors.disciplines) setErrors({ ...errors, disciplines: undefined });
                  }}
                  error={errors.disciplines}
                />
              </Col>
            </Row>
            <Row>
              {!isEditing && (
                <Col xs={loading ? 9 : 12}>
                  <FileUpload 
                    setFile={setFile} 
                    setThumbnail={setThumbnail}
                    error={errors.file}
                    onFileSelect={() => {
                      if (errors.file) setErrors({ ...errors, file: undefined });
                    }}
                  />
                </Col>
              )}
              {loading && (
                <Col xs={3} style={{ display: 'flex', justifyContent: 'center', alignItems: 'center'}}>
                  <Spinner />
                </Col>
              )}
            </Row>
          </Form>
        )}
      </Modal.Body>
      <Modal.Footer>
        {step > 1 && (
          <Button variant="danger" className={styles.buttonDanger} onClick={handleCancel}>
            Cancel
          </Button>
        )}
        <Button
          variant="dark"
          className={step === 3 ? styles.buttonDanger : styles.button}
          onClick={handleContinue}
        >
          {step === 2 ? 'Submit' : 'Continue'}
        </Button>
      </Modal.Footer>
    </Modal>
  );
};

export default VideoDashboardCreateModal;