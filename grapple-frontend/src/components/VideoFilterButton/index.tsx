import React, { useState, useEffect } from "react";
import { Dropdown } from "react-bootstrap";
import { CiFilter } from "react-icons/ci";
import { FaCheck } from "react-icons/fa";
import styles from './VideoFilterButton.module.css';
import { difficulty, disciplines } from "@/util/default-values";
import { useUpdateDisplaySeries } from "@/hook/series";

interface FilterOption {
  label: string;
  value: string;
}

const VideoFilterButton = ({
  isCoach = false,
}: any) => {
  const [formData, setFormData] = useState({
    difficulties: [] as FilterOption[],
    disciplines: [] as FilterOption[],
  });
  const querySeries = useUpdateDisplaySeries();

  useEffect(() => {
    runQuery();
  }, [formData]);

  const runQuery = () => {
    querySeries.mutate({
      ...(formData.difficulties.length > 0 ? { difficulty: formData.difficulties.map(d => d.value) } : {}),
      ...(formData.disciplines.length > 0 ? { discipline: formData.disciplines.map(d => d.label) } : {}),
      page_size: 6,
    } as any);
  };

  const handleFilterSelect = (type: 'difficulties' | 'disciplines', option: FilterOption) => {
    setFormData(prev => ({
      ...prev,
      [type]: prev[type].some(item => item.value === option.value)
        ? prev[type].filter(item => item.value !== option.value)
        : [...prev[type], option]
    }));
  };

  const isSelected = (type: 'difficulties' | 'disciplines', value: string) => {
    return formData[type].some(item => item.value === value);
  };

  const getItemStyle = (isSelected: boolean) => ({
    backgroundColor: isSelected ? 'black' : 'white',
    color: isSelected ? 'white' : 'black',
    fontWeight: isSelected ? 'bold' : 'normal',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '8px 12px',
    transition: 'background-color 0.3s, color 0.3s',
  });

  const clearFilters = () => {
    setFormData({ difficulties: [], disciplines: [] });
  };

  const getSelectedFiltersText = () => {
    const allFilters = [...formData.difficulties, ...formData.disciplines];
    return allFilters.length > 0 
      ? allFilters.map(filter => filter.label).join(', ')
      : 'Filter';
  };

  return (
    <div>
      <Dropdown>
        <Dropdown.Toggle variant="white" id="filter-dropdown" className={styles.filterButton}>
          <CiFilter style={{ marginRight: 10 }} />
          <span style={{ marginRight: 10 }}>
            {getSelectedFiltersText()}
          </span>
        </Dropdown.Toggle>
        <Dropdown.Menu className={styles.dropdownMenu}>
          <Dropdown drop="start">
            <Dropdown.Toggle variant="white" id="difficulty-dropdown">
              <span style={{ marginLeft: 10 }}>Filter by Difficulty</span>
            </Dropdown.Toggle>
            <Dropdown.Menu>
              {difficulty.map((diff, index) => (
                <Dropdown.Item
                  key={index}
                  onClick={() => handleFilterSelect('difficulties', diff)}
                  style={getItemStyle(isSelected('difficulties', diff.value))}
                >
                  <span>{diff.label}</span>
                  {isSelected('difficulties', diff.value) && <FaCheck color="white" />}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          </Dropdown>
          <Dropdown drop="start">
            <Dropdown.Toggle variant="white" id="discipline-dropdown">
              <span style={{ marginLeft: 10 }}>Filter by Discipline</span>
            </Dropdown.Toggle>
            <Dropdown.Menu>
              {disciplines.map((disc, index) => (
                <Dropdown.Item
                  key={index}
                  onClick={() => handleFilterSelect('disciplines', disc)}
                  style={getItemStyle(isSelected('disciplines', disc.value))}
                >
                  <span>{disc.label}</span>
                  {isSelected('disciplines', disc.value) && <FaCheck color="white" />}
                </Dropdown.Item>
              ))}
            </Dropdown.Menu>
          </Dropdown>
          <Dropdown.Item href="#/bookmarked">Bookmarked</Dropdown.Item>
          <Dropdown.Divider />
          <Dropdown.Item onClick={clearFilters}>Clear</Dropdown.Item>
        </Dropdown.Menu>
      </Dropdown>
    </div>
  );
};

export default VideoFilterButton;