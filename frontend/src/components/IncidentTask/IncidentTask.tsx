import { ChangeEvent, useEffect, useRef, useState } from 'react';
import { ChevronIcon, UserIcon } from '../Icons/Icons';
import { getIntegrationIcon, handleKeyDown } from '../../utils/utils';
import {
  IIncidentTask,
  Incident
} from '../../contexts/IncidentsProvider/IncidentsProvider';
import { toast } from 'react-toastify';
import { useAuthContext } from '../../hooks/useAuthContext';
import { useIncidentsContext } from '../../hooks/useIncidentsContext';
import Button from '../Button/Button';
import classNames from 'classnames';
import Input from '../Input/Input';
import TextArea from '../TextArea/TextArea';
import './IncidentTask.scss';

interface AddCommentResponse {
  updatedIncident: Incident;
}

interface IncidentTaskProps {
  incidentTask: IIncidentTask;
}

interface TaskStatusResponse {
  updatedIncident: Incident;
}

const IncidentTask = ({ incidentTask }: IncidentTaskProps) => {
  const [commentContent, setCommentContent] = useState('');
  const [commentTitle, setCommentTitle] = useState('');
  const [isCommentEditorOpen, setIsCommentEditorOpen] = useState(false);
  const [isOpen, setIsOpen] = useState(false);

  const { namespaceId } = useAuthContext();
  const { selectedIncident, setSelectedIncident } = useIncidentsContext();

  const abortControllerRef = useRef(new AbortController());
  const commentEditorTitleInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isCommentEditorOpen) {
      commentEditorTitleInputRef.current?.focus();
    }
  }, [isCommentEditorOpen]);

  const handleCloseCommentEditor = () => {
    setCommentTitle('');
    setCommentContent('');
    setIsCommentEditorOpen(false);
  };

  const handleCommentContentChange = (
    e: React.ChangeEvent<HTMLTextAreaElement>
  ) => setCommentContent(e.target.value);

  const handleCommentTitleChange = (e: React.ChangeEvent<HTMLInputElement>) =>
    setCommentTitle(e.target.value);

  const handleOpenCommentEditor = () => setIsCommentEditorOpen(true);

  const handleSaveComment = async (incidentItem: IIncidentTask) => {
    if (!commentTitle || !commentContent) return;

    try {
      const response = await fetch(
        `${import.meta.env.VITE_SERVER_API_URL}/${namespaceId}/incident/${selectedIncident?.id}/${incidentItem.id}/add-task-comment`,
        {
          body: JSON.stringify({
            content: commentContent,
            title: commentTitle
          }),
          headers: {
            'Content-Type': 'application/json'
          },
          method: 'POST'
        }
      );

      if (!response.ok) throw new Error('Failed to save the comment');

      const data: AddCommentResponse = await response.json();

      setSelectedIncident(data.updatedIncident);
      setCommentTitle('');
      setCommentContent('');
      setIsCommentEditorOpen(false);
      toast.success('Comment saved successfully');
    } catch (error) {
      if (error instanceof Error) {
        toast.error(error.message);
      } else {
        toast.error('An error occurred while saving the comment');
      }
    }
  };

  const handleStatusChange = async (e: ChangeEvent<HTMLInputElement>) => {
    const updatedTaskStatus = e.target.checked;

    abortControllerRef.current.abort();
    abortControllerRef.current = new AbortController();

    try {
      const response = await fetch(
        `${import.meta.env.VITE_SERVER_API_URL}/${namespaceId}/incident/${selectedIncident?.id}/${incidentTask.id}/status`,
        {
          body: JSON.stringify({
            updatedTaskStatus
          }),
          headers: {
            'Content-Type': 'application/json'
          },
          method: 'PATCH',
          signal: abortControllerRef.current.signal
        }
      );

      if (!response.ok) throw new Error('Failed to update task status');

      const data: TaskStatusResponse = await response.json();

      setSelectedIncident(data.updatedIncident);
      toast.success('Task status updated successfully');
    } catch (error) {
      if (error instanceof Error) {
        if (error.name !== 'AbortError') toast.error(error.message);
      } else {
        toast.error('An error occurred while updating the task status');
      }
    }
  };

  const handleToggleOpen = () => setIsOpen(prev => !prev);

  return (
    <li className="incident-task-container">
      <div className="incident-task-tile">
        <div className="incident-task-tile-left">
          <span className="incident-task-drag-handle">
            <span className="dot" />
            <span className="dot" />
            <span className="dot" />
          </span>
          <ChevronIcon
            className={classNames('arrow-icon', {
              open: isOpen
            })}
            height={16}
            onClick={handleToggleOpen}
            onKeyDown={handleKeyDown(handleToggleOpen)}
            tabIndex={0}
            width={16}
          />
          <span className="incident-task-name" onClick={handleToggleOpen}>
            {incidentTask.taskName}
          </span>
        </div>
        <div className="incident-task-tile-right">
          <label
            className={classNames('incident-task-checkbox', {
              checked: incidentTask.isDone
            })}
            htmlFor={incidentTask.taskName}
          >
            <input
              className="incident-task-checkbox-input"
              defaultChecked={incidentTask.isDone}
              onChange={handleStatusChange}
              id={incidentTask.taskName}
              type="checkbox"
            />
            <span className="incident-task-checkbox-label">Done</span>
          </label>
          <span className="incident-task-assignee">
            {incidentTask.assignee.name ? (
              <img
                alt={`User: ${incidentTask.assignee.name}`}
                src={incidentTask.assignee.photoUrl}
              />
            ) : (
              <UserIcon height={16} width={16} />
            )}
          </span>
        </div>
      </div>
      {isOpen && (
        <div className="incident-task-items">
          {incidentTask?.items?.map((incidentItem, index) => (
            <div
              className="incident-task-item"
              key={`${incidentItem.source}-${index}`}
            >
              <div className="incident-task-item-source">
                {getIntegrationIcon(incidentItem.source)}
              </div>
              <div className="incident-task-item-content">
                {incidentItem.content?.map(item => (
                  <div
                    className="incident-task-item-content-info"
                    key={`${item.key}-${item.value}`}
                  >
                    <span className="incident-task-item-content-key">
                      {item.key}:
                    </span>
                    <span className="incident-task-item-content-value">
                      {item.value}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          ))}
          <div className="users-comments-container">
            <h3 className="users-comments-header-title">Users Comments</h3>
            <hr className="users-comments-separator" />
            {incidentTask.comments.length > 0 && (
              <div className="users-comments-body">
                {incidentTask.comments.map((comment, index) => (
                  <div className="user-comment-container" key={index}>
                    <div
                      className="user-comment-content-info"
                      key={`${comment.content.key}-${comment.content.value}`}
                    >
                      <span className="user-comment-content-key">
                        {comment.content.key}
                      </span>
                      <span className="user-comment-content-value">
                        {comment.content.value}
                      </span>
                    </div>
                    <i className="user-comment-author-info">
                      <span className="user-comment-author-name">
                        ~ {comment.source.name}
                      </span>
                      <span className="user-comment-date">
                        {new Date(comment.timestamp * 1000).toLocaleDateString(
                          'en-US',
                          {
                            day: '2-digit',
                            month: 'short',
                            year: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit'
                          }
                        )}
                      </span>
                    </i>
                  </div>
                ))}
              </div>
            )}
          </div>
          {isCommentEditorOpen && (
            <div className="comment-editor-container">
              <Input
                label="Comment Title"
                onChange={handleCommentTitleChange}
                placeholder="Enter comment title"
                ref={commentEditorTitleInputRef}
                type="text"
                value={commentTitle}
              />
              <TextArea
                label="Comment Content"
                onChange={handleCommentContentChange}
                placeholder="Type your comment here..."
                value={commentContent}
              />
            </div>
          )}
          <div className="comment-editor-buttons">
            {isCommentEditorOpen && (
              <>
                <Button onClick={handleCloseCommentEditor}>
                  Discard Comment
                </Button>
                <Button
                  aria-disabled={!commentTitle || !commentContent}
                  disabled={!commentTitle || !commentContent}
                  onClick={() => handleSaveComment(incidentTask)}
                >
                  Save Comment
                </Button>
              </>
            )}
            {!isCommentEditorOpen && (
              <Button onClick={handleOpenCommentEditor}>Add Comment</Button>
            )}
          </div>
        </div>
      )}
    </li>
  );
};

export default IncidentTask;