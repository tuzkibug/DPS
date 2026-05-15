import React from 'react';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import { TaskCreatePage } from './pages/TaskCreatePage';
import { TaskDetailPage } from './pages/TaskDetailPage';
import { TaskListPage } from './pages/TaskListPage';
import { Layout, Menu } from 'antd';

const { Header, Content, Footer } = Layout;

const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Layout className="layout" style={{ minHeight: '100vh' }}>
        <Header>
          <div style={{ float: 'left', color: 'white', fontSize: 20, marginRight: 40 }}>
            DPS - DNS Packets Sender
          </div>
          <Menu theme="dark" mode="horizontal" defaultSelectedKeys={['tasks']}>
            <Menu.Item key="tasks">
              <Link to="/tasks">Tasks</Link>
            </Menu.Item>
            <Menu.Item key="create">
              <Link to="/tasks/new">Create Task</Link>
            </Menu.Item>
          </Menu>
        </Header>
        <Content style={{ padding: 24 }}>
          <Routes>
            <Route path="/" element={<TaskListPage />} />
            <Route path="/tasks" element={<TaskListPage />} />
            <Route path="/tasks/new" element={<TaskCreatePage />} />
            <Route path="/tasks/:id" element={<TaskDetailPage />} />
          </Routes>
        </Content>
        <Footer style={{ textAlign: 'center', color: '#999' }}>
          Created by Tuzkibug
        </Footer>
      </Layout>
    </BrowserRouter>
  );
};

export default App;